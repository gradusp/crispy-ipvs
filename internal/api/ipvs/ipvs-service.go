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

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/gradusp/crispy-ipvs/pkg/ipvs"
	ipvsAdm "github.com/gradusp/crispy-ipvs/pkg/net/ipvs"
	"github.com/gradusp/go-platform/logger"
	"github.com/gradusp/go-platform/pkg/jsonview"
	"github.com/gradusp/go-platform/pkg/parallel"
	"github.com/gradusp/go-platform/server"
	grpcRt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
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

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Bool("include-reals", req.GetIncludeReals()),
	)

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
	span := trace.SpanFromContext(ctx)

	identity := req.GetVirtualServerIdentity()
	span.SetAttributes(
		attribute.Stringer("virtual-server", jsonview.Stringer(identity)),
		attribute.Bool("include-reals", req.GetIncludeReals()),
	)
	var conv VirtualServerIdentityConv
	if err = conv.FromPb(identity); err != nil {
		err = srv.errWithDetails(codes.InvalidArgument, err.Error(), identity)
		return
	}
	errSuccess := errors.New("1")
	err = srv.admin.ListVirtualServers(ctx, func(vs ipvsAdm.VirtualServer) error {
		if !ipvsAdm.IsIdentitiesEq(vs.Identity, conv.Identity) {
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
		err = status.Errorf(codes.NotFound, "virtual-server %v is not found", conv.Identity)
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
	forceUpsert := req.GetForceUpsert()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int("delete-count", len(req.GetDelete())),
		attribute.Int("update-count", len(req.GetUpdate())),
		attribute.Bool("force-upsert", forceUpsert),
	)

	resp = new(ipvs.UpdateVirtualServersResponse)
	if del := req.GetDelete(); len(del) > 0 {
		srv.addSpanDbgEvent(ctx, span, "delete", trace.WithAttributes(
			attribute.Stringer("virtual-servers", jsonview.Stringer(del)),
		))
		err = parallel.ExecAbstract(len(del), 10, func(i int) error {
			toDel := del[i]
			if whenSeen(toDel) {
				return nil
			}
			issue, e := srv.delVS(ctx, toDel)
			if e != nil {
				return e
			}
			if issue != nil {
				mx.Lock()
				resp.Issues = append(resp.Issues, issue)
				mx.Unlock()
			}
			return nil
		})
		if err != nil {
			return
		}
	}
	if upd := req.GetUpdate(); len(upd) > 0 {
		srv.addSpanDbgEvent(ctx, span, "update", trace.WithAttributes(
			attribute.Stringer("virtual-servers", jsonview.Stringer(upd)),
		))
		if forceUpsert && len(seen) > 0 {
			seen = make(map[string]bool)
		}
		err = parallel.ExecAbstract(len(upd), 10, func(i int) error {
			toUpd := upd[i]
			if whenSeen(toUpd) {
				return nil
			}
			issue, e := srv.updVS(ctx, toUpd, forceUpsert)
			if e != nil {
				return e
			}
			if issue != nil {
				mx.Lock()
				resp.Issues = append(resp.Issues, issue)
				mx.Unlock()
			}
			return nil
		})
	}
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
	forceUpsert := req.GetForceUpsert()
	vsID := req.GetVirtualServerIdentity()

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int("delete-count", len(req.GetDelete())),
		attribute.Int("update-count", len(req.GetUpdate())),
		attribute.Stringer("virtual-server", jsonview.Stringer(vsID)),
		attribute.Bool("force-upsert", forceUpsert),
	)

	var vsIDConv VirtualServerIdentityConv
	if err = vsIDConv.FromPb(vsID); err != nil {
		err = srv.errWithDetails(codes.InvalidArgument, err.Error(), vsID)
		return
	}

	resp = new(ipvs.UpdateRealServersResponse)
	if del := req.GetDelete(); len(del) > 0 {
		srv.addSpanDbgEvent(ctx, span, "delete", trace.WithAttributes(
			attribute.Stringer("real-servers", jsonview.Stringer(del)),
		))
		err = parallel.ExecAbstract(len(del), 10, func(i int) error {
			toDel := del[i]
			if whenSeen(toDel) {
				return nil
			}
			iss, e := srv.delRS(ctx, vsIDConv.Identity, toDel)
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
	}
	if upd := req.GetUpdate(); len(upd) > 0 {
		srv.addSpanDbgEvent(ctx, span, "update", trace.WithAttributes(
			attribute.Stringer("real-servers", jsonview.Stringer(upd)),
		))
		if forceUpsert && len(seen) > 0 {
			seen = make(map[string]bool)
		}
		err = parallel.ExecAbstract(len(upd), 10, func(i int) error {
			toUpd := upd[i]
			if whenSeen(toUpd.Address) {
				return nil
			}
			iss, e := srv.updRS(ctx, vsIDConv.Identity, toUpd, forceUpsert)
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
	}
	return resp, err
}

func (srv *ipvsAdminSrv) delRS(ctx context.Context, identity ipvsAdm.VirtualServerIdentity, toDel *ipvs.RealServerAddress) (*ipvs.RealServerIssue, error) {
	var rs AddressConv
	rs.FromPb(toDel)
	err := srv.admin.RemoveRealServer(ctx, identity, rs.Address)
	if err == nil {
		return nil, nil
	}
	var issue *ipvs.RealServerIssue
	if reason := srv.ifReason(err); reason != nil {
		err = nil
		issue = &ipvs.RealServerIssue{
			When: &ipvs.RealServerIssue_Delete{
				Delete: &ipvs.RealServerAddress{
					Host: toDel.GetHost(),
					Port: toDel.GetPort(),
				},
			},
			Reason: reason,
		}
	}
	return issue, err
}

func (srv *ipvsAdminSrv) updRS(ctx context.Context, identity ipvsAdm.VirtualServerIdentity, toUpd *ipvs.RealServer, forceUpsert bool) (*ipvs.RealServerIssue, error) {
	var rs RealServerConv
	err := rs.FromPb(toUpd)
	if err != nil {
		return nil, srv.errWithDetails(codes.InvalidArgument, err.Error(), toUpd)
	}
	var opts []ipvsAdm.AdminOption
	if forceUpsert {
		opts = append(opts, ipvsAdm.ForceAddIfNotExist{})
	}
	if err = srv.admin.UpdateRealServer(ctx, identity, rs.RealServer, opts...); err == nil {
		return nil, nil
	}

	var issue *ipvs.RealServerIssue
	if reason := srv.ifReason(err); reason != nil {
		err = nil
		issue = &ipvs.RealServerIssue{
			When: &ipvs.RealServerIssue_Update{
				Update: toUpd,
			},
			Reason: reason,
		}
	}
	return issue, err
}

func (srv *ipvsAdminSrv) updVS(ctx context.Context, toUpd *ipvs.VirtualServer, forceUpsert bool) (*ipvs.VirtualServerIssue, error) {
	var vsConv VirtualServerConv
	err := vsConv.FromPb(toUpd)
	if err != nil {
		return nil, srv.errWithDetails(codes.InvalidArgument, err.Error(), toUpd)
	}
	var opts []ipvsAdm.AdminOption
	if forceUpsert {
		opts = append(opts, ipvsAdm.ForceAddIfNotExist{})
	}
	if err = srv.admin.UpdateVirtualServer(ctx, vsConv.VirtualServer, opts...); err == nil {
		return nil, nil
	}
	var issue *ipvs.VirtualServerIssue
	if reason := srv.ifReason(err); reason != nil {
		err = nil
		issue = &ipvs.VirtualServerIssue{
			When:   &ipvs.VirtualServerIssue_Update{Update: toUpd},
			Reason: reason,
		}
	}
	return issue, err
}

func (srv *ipvsAdminSrv) delVS(ctx context.Context, toDel *ipvs.VirtualServerIdentity) (*ipvs.VirtualServerIssue, error) {
	var identityConv VirtualServerIdentityConv
	err := identityConv.FromPb(toDel)
	if err != nil {
		return nil, srv.errWithDetails(codes.InvalidArgument, err.Error(), toDel)
	}
	if err = srv.admin.RemoveVirtualServer(ctx, identityConv.Identity); err == nil {
		return nil, nil
	}
	var issue *ipvs.VirtualServerIssue
	if reason := srv.ifReason(err); reason != nil {
		err = nil
		issue = &ipvs.VirtualServerIssue{
			When:   &ipvs.VirtualServerIssue_Delete{Delete: toDel},
			Reason: reason,
		}
	}
	return issue, err
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

func (srv *ipvsAdminSrv) ifReason(err error) *ipvs.IssueReason {
	if err == nil {
		return nil
	}
	var reason *ipvs.IssueReason
	if errors.Is(err, ipvsAdm.ErrVirtualServerNotExist) {
		reason = &ipvs.IssueReason{
			Code: ipvs.IssueReason_VirtualServerNotFound,
		}
	} else if errors.Is(err, ipvsAdm.ErrUnsupported) {
		reason = &ipvs.IssueReason{
			Code: ipvs.IssueReason_Unsupported,
		}
	} else if errors.Is(err, ipvsAdm.ErrRealServerNotExist) {
		reason = &ipvs.IssueReason{
			Code: ipvs.IssueReason_RealServerNotFound,
		}
	} else if errors.Is(err, ipvsAdm.ErrExternal) {
		reason = &ipvs.IssueReason{
			Code: ipvs.IssueReason_ExternalError,
		}
	}
	if reason != nil {
		reason.Message = err.Error()
	}
	return reason
}

func (srv *ipvsAdminSrv) errWithDetails(code codes.Code, msg string, details ...proto.Message) error {
	stat := status.New(code, msg)
	if len(details) > 0 {
		if stat2, e := stat.WithDetails(details...); e == nil {
			return stat2.Err()
		}
	}
	return stat.Err()
}
