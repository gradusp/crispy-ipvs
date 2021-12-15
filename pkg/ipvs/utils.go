package ipvs

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	//String2NetworkTransport String to NetworkTransport
	String2NetworkTransport = make(map[string]NetworkTransport)
	//NetworkTransport2String NetworkTransport to String
	NetworkTransport2String = make(map[NetworkTransport]string)

	//String2ScheduleMethod String to ScheduleMethod
	String2ScheduleMethod = make(map[string]ScheduleMethod)
	//ScheduleMethod2String ScheduleMethod to String
	ScheduleMethod2String = make(map[ScheduleMethod]string)

	//PacketFwdMethod2String PacketFwdMethod to String
	PacketFwdMethod2String = make(map[PacketFwdMethod]string)
	//String2PacketFwdMethod String to PacketFwdMethod
	String2PacketFwdMethod = make(map[string]PacketFwdMethod)
)

type protoEnumWalker struct {
	protoreflect.EnumDescriptor
}

func (walker protoEnumWalker) traverseValues(extInfo *protoimpl.ExtensionInfo,
	f func(value protoreflect.EnumNumber, valueExt interface{})) {

	vals := walker.EnumDescriptor.Values()
	for i := 0; i < vals.Len(); i++ {
		v := vals.Get(i)
		var valueExt interface{}
		if extInfo != nil {
			switch o := v.Options().(type) {
			case *descriptorpb.EnumValueOptions:
				valueExt = proto.GetExtension(o, extInfo)
			}
		}
		f(v.Number(), valueExt)
	}
}

func init() {
	protoEnumWalker{EnumDescriptor: NetworkTransport(0).Descriptor()}.
		traverseValues(E_Transport,
			func(value protoreflect.EnumNumber, valueExt interface{}) {
				switch s := valueExt.(type) {
				case string:
					String2NetworkTransport[s] = NetworkTransport(value)
					NetworkTransport2String[NetworkTransport(value)] = s
				}
			},
		)

	protoEnumWalker{EnumDescriptor: ScheduleMethod(0).Descriptor()}.
		traverseValues(E_ScheduleAlg,
			func(value protoreflect.EnumNumber, valueExt interface{}) {
				switch s := valueExt.(type) {
				case string:
					ScheduleMethod2String[ScheduleMethod(value)] = s
					String2ScheduleMethod[s] = ScheduleMethod(value)
				}
			},
		)

	protoEnumWalker{EnumDescriptor: PacketFwdMethod(0).Descriptor()}.
		traverseValues(E_FwdAlg,
			func(value protoreflect.EnumNumber, valueExt interface{}) {
				switch s := valueExt.(type) {
				case string:
					PacketFwdMethod2String[PacketFwdMethod(value)] = s
					String2PacketFwdMethod[s] = PacketFwdMethod(value)
				}
			},
		)
}
