package ipvs

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"strconv"
	"sync"

	"github.com/gradusp/crispy-ipvs/pkg/ipvs"
	ipvsAdm "github.com/gradusp/crispy-ipvs/pkg/net/ipvs"
	"github.com/gradusp/go-platform/logger"
	"github.com/gradusp/go-platform/pkg/parallel"
	"github.com/gradusp/go-platform/server"
	grpcRt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//NewIpvsAdminService creates roure service
func NewIpvsAdminService(ctx context.Context, adm ipvsAdm.Admin) server.APIService {
	ret := &ipvsAdminSrv{
		appCtx: ctx,
		sema:   make(chan struct{}, 1),
		admin:  adm,
	}
	runtime.SetFinalizer(ret, func(o *ipvsAdminSrv) {
		close(o.sema)
	})
	return ret
}

//GetSwaggerDocs get swagger spec docs
func GetSwaggerDocs() (*server.SwaggerSpec, error) {
	const api = "ipvs/GetSwaggerDocs"
	ret := new(server.SwaggerSpec)
	err := json.Unmarshal(rawSwagger, ret)
	return ret, errors.Wrap(err, api)
}

var (
	_ ipvs.IpvsAdminServer   = (*ipvsAdminSrv)(nil)
	_ server.APIService      = (*ipvsAdminSrv)(nil)
	_ server.APIGatewayProxy = (*ipvsAdminSrv)(nil)

	//go:embed ipvs.swagger.json
	rawSwagger []byte
)

type ipvsAdminSrv struct {
	ipvs.UnimplementedIpvsAdminServer
	appCtx context.Context
	admin  ipvsAdm.Admin
	sema   chan struct{}
}

//Description impl server.APIService
func (srv *ipvsAdminSrv) Description() grpc.ServiceDesc {
	return ipvs.IpvsAdmin_ServiceDesc
}

//RegisterGRPC impl server.APIService
func (srv *ipvsAdminSrv) RegisterGRPC(_ context.Context, s *grpc.Server) error {
	ipvs.RegisterIpvsAdminServer(s, srv)
	return nil
}

//RegisterProxyGW impl server.APIGatewayProxy
func (srv *ipvsAdminSrv) RegisterProxyGW(ctx context.Context, mux *grpcRt.ServeMux, c *grpc.ClientConn) error {
	return ipvs.RegisterIpvsAdminHandler(ctx, mux, c)
}

//ListVirtualServers impl service
func (srv *ipvsAdminSrv) ListVirtualServers(ctx context.Context, req *ipvs.ListVirtualServersRequest) (resp *ipvs.ListVirtualServersResponse, err error) {
	defer func() {
		err = srv.correctError(err)
	}()

	type (
		itemT = *ipvs.VirtualServerWithReals
		keyT  = ipvsAdm.VirtualServerIdentity
		mT    = struct {
			keyT
			itemT
		}
	)
	var ids []mT
	includeReals := req.GetIncludeReals()
	resp = new(ipvs.ListVirtualServersResponse)
	err = srv.admin.ListVirtualServers(ctx, func(vs ipvsAdm.VirtualServer) error {
		v, e := VirtualServerConv{VirtualServer: vs}.ToPb()
		if e != nil {
			return e
		}
		item := &ipvs.VirtualServerWithReals{
			VirtualServer: v,
		}
		resp.VirtualServers = append(resp.VirtualServers, item)
		if includeReals {
			ids = append(ids, mT{keyT: vs.Identity, itemT: item})
		}
		return nil
	})
	if err != nil {
		return
	}
	err = parallel.ExecAbstract(len(ids), 10, func(i int) error {
		k := ids[i]
		item := k.itemT
		return srv.admin.ListRealServers(ctx, k.keyT, func(rs ipvsAdm.RealServer) error {
			c, e := RealServerConv{RealServer: rs}.ToPb()
			if e != nil {
				return e
			}
			item.RealServers = append(item.RealServers, c)
			return nil
		})
	})

	return resp, err
}

//FindVirtualServer impl service
func (srv *ipvsAdminSrv) FindVirtualServer(ctx context.Context, req *ipvs.FindVirtualServerRequest) (resp *ipvs.FindVirtualServerResponse, err error) {
	defer func() {
		err = srv.correctError(err)
	}()
	identity := req.GetVirtualServerIdentity()
	var conv VirtualServerIdentityConv
	if err = conv.FromPb(identity); err != nil {
		return
	}

	errSuccess := errors.New("1")
	err = srv.admin.ListVirtualServers(ctx, func(vs ipvsAdm.VirtualServer) error {
		if !(vs.Identity == conv) {
			return nil
		}
		var e error
		resp = &ipvs.FindVirtualServerResponse{
			VirtualServer: new(ipvs.VirtualServerWithReals),
		}
		resp.VirtualServer.VirtualServer, e = VirtualServerConv{VirtualServer: vs}.ToPb()
		if e == nil && req.GetIncludeReals() {
			var reals []*ipvs.RealServer
			e = srv.admin.ListRealServers(ctx, vs.Identity, func(rs ipvsAdm.RealServer) error {
				r, e2 := RealServerConv{RealServer: rs}.ToPb()
				if e2 == nil {
					reals = append(reals, r)
				}
				return e2
			})
			resp.VirtualServer.RealServers = reals
		}
		if e == nil {
			e = errSuccess
		}
		return e
	})
	if err != nil {
		if errors.Is(err, errSuccess) {
			err = nil
		}
		return
	}
	if resp == nil {
		err = status.Errorf(codes.NotFound, "virtual-server(%v) is not found", conv.VirtualServerIdentity)
	}
	return nil, err
}

//UpdateVirtualServers impl service
func (srv *ipvsAdminSrv) UpdateVirtualServers(ctx context.Context, req *ipvs.UpdateVirtualServersRequest) (resp *ipvs.UpdateVirtualServersResponse, err error) {
	var leave func()
	if leave, err = srv.enter(ctx); err != nil {
		return
	}
	defer func() {
		leave()
		err = srv.correctError(err)
	}()
	var mx sync.Mutex
	seen := make(map[string]bool)
	whenSeen := func(s fmt.Stringer) bool {
		mx.Lock()
		defer mx.Unlock()
		a := s.String()
		ret := seen[a]
		if !ret {
			seen[a] = true
		}
		return ret
	}
	del := req.GetDelete()
	resp = new(ipvs.UpdateVirtualServersResponse)
	err = parallel.ExecAbstract(len(del), 10, func(i int) error {
		toDel := del[i]
		if whenSeen(toDel) {
			return nil
		}
		iss, e := srv.delVS(ctx, toDel)
		if e != nil {
			return e
		}
		if iss != nil {
			mx.Lock()
			resp.Issues = append(resp.Issues, iss)
			mx.Unlock()
		}
		return nil
	})
	if err != nil {
		return
	}
	upd := req.GetUpdate()
	forceUpsert := req.GetForceUpsert()
	err = parallel.ExecAbstract(len(upd), 10, func(i int) error {
		toUpd := upd[i]
		if whenSeen(toUpd) {
			return nil
		}
		iss, e := srv.updVS(ctx, toUpd, forceUpsert)
		if e != nil {
			return e
		}
		if iss != nil {
			mx.Lock()
			resp.Issues = append(resp.Issues, iss)
			mx.Unlock()
		}
		return nil
	})
	return resp, err
}

//UpdateRealServers impl service
func (srv *ipvsAdminSrv) UpdateRealServers(ctx context.Context, req *ipvs.UpdateRealServersRequest) (resp *ipvs.UpdateRealServersResponse, err error) {
	var leave func()
	if leave, err = srv.enter(ctx); err != nil {
		return
	}
	defer func() {
		leave()
		err = srv.correctError(err)
	}()

	var mx sync.Mutex
	seen := make(map[string]bool)
	whenSeen := func(a *ipvs.RealServerAddress) bool {
		s := net.JoinHostPort(a.GetHost(), strconv.Itoa(int(a.GetPort())))
		mx.Lock()
		defer mx.Unlock()
		ret := seen[s]
		if !ret {
			seen[s] = true
		}
		return ret
	}
	resp = new(ipvs.UpdateRealServersResponse)
	vsID := req.GetVirtualServerIdentity()
	del := req.GetDelete()
	err = parallel.ExecAbstract(len(del), 10, func(i int) error {
		toDel := del[i]
		if whenSeen(toDel) {
			return nil
		}
		iss, e := srv.delRS(ctx, vsID, toDel)
		if e != nil {
			return e
		}
		if iss != nil {
			mx.Lock()
			resp.Issues = append(resp.Issues, iss)
			mx.Unlock()
		}
		return nil
	})
	if err != nil {
		return
	}
	upd := req.GetUpdate()
	forceUpsert := req.GetForceUpsert()
	err = parallel.ExecAbstract(len(upd), 10, func(i int) error {
		toUpd := upd[i]
		if whenSeen(toUpd.Address) {
			return nil
		}
		iss, e := srv.updRS(ctx, vsID, toUpd, forceUpsert)
		if e != nil {
			return e
		}
		if iss != nil {
			mx.Lock()
			resp.Issues = append(resp.Issues, iss)
			mx.Unlock()
		}
		return nil
	})
	return resp, err
}

func (srv *ipvsAdminSrv) delRS(ctx context.Context, vsID *ipvs.VirtualServerIdentity, toDel *ipvs.RealServerAddress) (*ipvs.RealServerIssue, error) {
	var rs AddressConv
	var vs VirtualServerIdentityConv
	rs.FromPb(toDel)
	err := vs.FromPb(vsID)
	if err != nil {
		return nil, err
	}
	if err = srv.admin.RemoveRealServer(ctx, vs, rs.Address); err == nil {
		return nil, nil
	}
	var reason *ipvs.IssueReason
	if errors.Is(err, ipvsAdm.ErrVirtualServerNotExist) {
		err = nil
		reason = &ipvs.IssueReason{
			Code:    ipvs.IssueReason_VirtualServerNotFound,
			Message: ipvs.IssueReason_VirtualServerNotFound.String(),
		}
	} else if errors.Is(err, ipvsAdm.ErrRealServerNotExist) {
		err = nil
		reason = &ipvs.IssueReason{
			Code:    ipvs.IssueReason_RealServerNotFound,
			Message: ipvs.IssueReason_RealServerNotFound.String(),
		}
	}
	if err != nil {
		return nil, err
	}
	issue := &ipvs.RealServerIssue{
		When: &ipvs.RealServerIssue_Delete{
			Delete: &ipvs.RealServerAddress{
				Host: toDel.GetHost(),
				Port: toDel.GetPort(),
			},
		},
		Reason: reason,
	}
	return issue, nil
}

func (srv *ipvsAdminSrv) updRS(ctx context.Context, vsID *ipvs.VirtualServerIdentity, toUpd *ipvs.RealServer, forceUpsert bool) (*ipvs.RealServerIssue, error) {
	var vs VirtualServerIdentityConv
	var rs RealServerConv

	err := vs.FromPb(vsID)
	if err != nil {
		return nil, err
	}
	if err = rs.FromPb(toUpd); err != nil {
		return nil, err
	}
	var opts []ipvsAdm.AdminOption
	if forceUpsert {
		opts = append(opts, ipvsAdm.ForceAddIfNotExist{})
	}
	if err = srv.admin.UpdateRealServer(ctx, vs, rs.RealServer, opts...); err == nil {
		return nil, nil
	}

	var reason *ipvs.IssueReason
	if errors.Is(err, ipvsAdm.ErrVirtualServerNotExist) {
		err = nil
		reason = &ipvs.IssueReason{
			Code:    ipvs.IssueReason_VirtualServerNotFound,
			Message: ipvs.IssueReason_VirtualServerNotFound.String(),
		}
	} else if errors.Is(err, ipvsAdm.ErrRealServerNotExist) {
		err = nil
		reason = &ipvs.IssueReason{
			Code:    ipvs.IssueReason_RealServerNotFound,
			Message: ipvs.IssueReason_RealServerNotFound.String(),
		}
	}
	if err != nil {
		return nil, err
	}
	issue := &ipvs.RealServerIssue{
		When: &ipvs.RealServerIssue_Update{
			Update: toUpd,
		},
		Reason: reason,
	}
	return issue, nil
}

func (srv *ipvsAdminSrv) updVS(ctx context.Context, toUpd *ipvs.VirtualServer, forceUpsert bool) (*ipvs.VirtualServerIssue, error) {
	var vsConv VirtualServerConv
	err := vsConv.FromPb(toUpd)
	if err != nil {
		return nil, err
	}
	var opts []ipvsAdm.AdminOption
	if forceUpsert {
		opts = append(opts, ipvsAdm.ForceAddIfNotExist{})
	}
	if err = srv.admin.UpdateVirtualSerer(ctx, vsConv.VirtualServer, opts...); err == nil {
		return nil, nil
	}
	var issue *ipvs.VirtualServerIssue
	if errors.Is(err, ipvsAdm.ErrVirtualServerNotExist) {
		err = nil
		issue = &ipvs.VirtualServerIssue{
			When: &ipvs.VirtualServerIssue_Update{Update: toUpd},
			Reason: &ipvs.IssueReason{
				Code:    ipvs.IssueReason_VirtualServerNotFound,
				Message: ipvs.IssueReason_VirtualServerNotFound.String(),
			},
		}
	}
	return issue, err
}

func (srv *ipvsAdminSrv) delVS(ctx context.Context, toDel *ipvs.VirtualServerIdentity) (*ipvs.VirtualServerIssue, error) {
	var identityConv VirtualServerIdentityConv
	err := identityConv.FromPb(toDel)
	if err != nil {
		return nil, err
	}
	if err = srv.admin.RemoveVirtualSerer(ctx, identityConv); err == nil {
		return nil, nil
	}
	var iss *ipvs.VirtualServerIssue
	if errors.Is(err, ipvsAdm.ErrVirtualServerNotExist) {
		err = nil
		iss = &ipvs.VirtualServerIssue{
			When: &ipvs.VirtualServerIssue_Delete{Delete: toDel},
			Reason: &ipvs.IssueReason{
				Code:    ipvs.IssueReason_VirtualServerNotFound,
				Message: ipvs.IssueReason_VirtualServerNotFound.String(),
			},
		}
	}
	return iss, err
}

func (srv *ipvsAdminSrv) addSpanDbgEvent(ctx context.Context, span trace.Span, eventName string, opts ...trace.EventOption) { //nolint:unused
	if logger.IsLevelEnabled(ctx, zap.DebugLevel) {
		span.AddEvent(eventName, opts...)
	}
}

func (srv *ipvsAdminSrv) correctError(err error) error {
	if err != nil && status.Code(err) == codes.Unknown {
		switch errors.Cause(err) {
		case context.DeadlineExceeded:
			return status.New(codes.DeadlineExceeded, err.Error()).Err()
		case context.Canceled:
			return status.New(codes.Canceled, err.Error()).Err()
		default:
			if e := new(url.Error); errors.As(err, &e) {
				switch errors.Cause(e.Err) {
				case context.Canceled:
					return status.New(codes.Canceled, err.Error()).Err()
				case context.DeadlineExceeded:
					return status.New(codes.DeadlineExceeded, err.Error()).Err()
				default:
					if e.Timeout() {
						return status.New(codes.DeadlineExceeded, err.Error()).Err()
					}
				}
			}
			err = status.New(codes.Internal, err.Error()).Err()
		}
	}
	return err
}

func (srv *ipvsAdminSrv) enter(ctx context.Context) (leave func(), err error) {
	select {
	case <-srv.appCtx.Done():
		err = srv.appCtx.Err()
	case <-ctx.Done():
		err = ctx.Err()
	case srv.sema <- struct{}{}:
		var o sync.Once
		leave = func() {
			o.Do(func() {
				<-srv.sema
			})
		}
		return
	}
	err = status.FromContextError(err).Err()
	return
}
