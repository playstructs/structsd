// Code generated by protoc-gen-go-pulsar. DO NOT EDIT.
package structs

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	runtime "github.com/cosmos/cosmos-proto/runtime"
	_ "github.com/cosmos/gogoproto/gogoproto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	io "io"
	reflect "reflect"
	sync "sync"
)

var (
	md_Infusion                 protoreflect.MessageDescriptor
	fd_Infusion_destinationType protoreflect.FieldDescriptor
	fd_Infusion_destinationId   protoreflect.FieldDescriptor
	fd_Infusion_fuel            protoreflect.FieldDescriptor
	fd_Infusion_power           protoreflect.FieldDescriptor
	fd_Infusion_commission      protoreflect.FieldDescriptor
	fd_Infusion_playerId        protoreflect.FieldDescriptor
	fd_Infusion_address         protoreflect.FieldDescriptor
)

func init() {
	file_structs_structs_infusion_proto_init()
	md_Infusion = File_structs_structs_infusion_proto.Messages().ByName("Infusion")
	fd_Infusion_destinationType = md_Infusion.Fields().ByName("destinationType")
	fd_Infusion_destinationId = md_Infusion.Fields().ByName("destinationId")
	fd_Infusion_fuel = md_Infusion.Fields().ByName("fuel")
	fd_Infusion_power = md_Infusion.Fields().ByName("power")
	fd_Infusion_commission = md_Infusion.Fields().ByName("commission")
	fd_Infusion_playerId = md_Infusion.Fields().ByName("playerId")
	fd_Infusion_address = md_Infusion.Fields().ByName("address")
}

var _ protoreflect.Message = (*fastReflection_Infusion)(nil)

type fastReflection_Infusion Infusion

func (x *Infusion) ProtoReflect() protoreflect.Message {
	return (*fastReflection_Infusion)(x)
}

func (x *Infusion) slowProtoReflect() protoreflect.Message {
	mi := &file_structs_structs_infusion_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

var _fastReflection_Infusion_messageType fastReflection_Infusion_messageType
var _ protoreflect.MessageType = fastReflection_Infusion_messageType{}

type fastReflection_Infusion_messageType struct{}

func (x fastReflection_Infusion_messageType) Zero() protoreflect.Message {
	return (*fastReflection_Infusion)(nil)
}
func (x fastReflection_Infusion_messageType) New() protoreflect.Message {
	return new(fastReflection_Infusion)
}
func (x fastReflection_Infusion_messageType) Descriptor() protoreflect.MessageDescriptor {
	return md_Infusion
}

// Descriptor returns message descriptor, which contains only the protobuf
// type information for the message.
func (x *fastReflection_Infusion) Descriptor() protoreflect.MessageDescriptor {
	return md_Infusion
}

// Type returns the message type, which encapsulates both Go and protobuf
// type information. If the Go type information is not needed,
// it is recommended that the message descriptor be used instead.
func (x *fastReflection_Infusion) Type() protoreflect.MessageType {
	return _fastReflection_Infusion_messageType
}

// New returns a newly allocated and mutable empty message.
func (x *fastReflection_Infusion) New() protoreflect.Message {
	return new(fastReflection_Infusion)
}

// Interface unwraps the message reflection interface and
// returns the underlying ProtoMessage interface.
func (x *fastReflection_Infusion) Interface() protoreflect.ProtoMessage {
	return (*Infusion)(x)
}

// Range iterates over every populated field in an undefined order,
// calling f for each field descriptor and value encountered.
// Range returns immediately if f returns false.
// While iterating, mutating operations may only be performed
// on the current field descriptor.
func (x *fastReflection_Infusion) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	if x.DestinationType != 0 {
		value := protoreflect.ValueOfEnum((protoreflect.EnumNumber)(x.DestinationType))
		if !f(fd_Infusion_destinationType, value) {
			return
		}
	}
	if x.DestinationId != "" {
		value := protoreflect.ValueOfString(x.DestinationId)
		if !f(fd_Infusion_destinationId, value) {
			return
		}
	}
	if x.Fuel != uint64(0) {
		value := protoreflect.ValueOfUint64(x.Fuel)
		if !f(fd_Infusion_fuel, value) {
			return
		}
	}
	if x.Power != uint64(0) {
		value := protoreflect.ValueOfUint64(x.Power)
		if !f(fd_Infusion_power, value) {
			return
		}
	}
	if x.Commission != "" {
		value := protoreflect.ValueOfString(x.Commission)
		if !f(fd_Infusion_commission, value) {
			return
		}
	}
	if x.PlayerId != "" {
		value := protoreflect.ValueOfString(x.PlayerId)
		if !f(fd_Infusion_playerId, value) {
			return
		}
	}
	if x.Address != "" {
		value := protoreflect.ValueOfString(x.Address)
		if !f(fd_Infusion_address, value) {
			return
		}
	}
}

// Has reports whether a field is populated.
//
// Some fields have the property of nullability where it is possible to
// distinguish between the default value of a field and whether the field
// was explicitly populated with the default value. Singular message fields,
// member fields of a oneof, and proto2 scalar fields are nullable. Such
// fields are populated only if explicitly set.
//
// In other cases (aside from the nullable cases above),
// a proto3 scalar field is populated if it contains a non-zero value, and
// a repeated field is populated if it is non-empty.
func (x *fastReflection_Infusion) Has(fd protoreflect.FieldDescriptor) bool {
	switch fd.FullName() {
	case "structs.Infusion.destinationType":
		return x.DestinationType != 0
	case "structs.Infusion.destinationId":
		return x.DestinationId != ""
	case "structs.Infusion.fuel":
		return x.Fuel != uint64(0)
	case "structs.Infusion.power":
		return x.Power != uint64(0)
	case "structs.Infusion.commission":
		return x.Commission != ""
	case "structs.Infusion.playerId":
		return x.PlayerId != ""
	case "structs.Infusion.address":
		return x.Address != ""
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: structs.Infusion"))
		}
		panic(fmt.Errorf("message structs.Infusion does not contain field %s", fd.FullName()))
	}
}

// Clear clears the field such that a subsequent Has call reports false.
//
// Clearing an extension field clears both the extension type and value
// associated with the given field number.
//
// Clear is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_Infusion) Clear(fd protoreflect.FieldDescriptor) {
	switch fd.FullName() {
	case "structs.Infusion.destinationType":
		x.DestinationType = 0
	case "structs.Infusion.destinationId":
		x.DestinationId = ""
	case "structs.Infusion.fuel":
		x.Fuel = uint64(0)
	case "structs.Infusion.power":
		x.Power = uint64(0)
	case "structs.Infusion.commission":
		x.Commission = ""
	case "structs.Infusion.playerId":
		x.PlayerId = ""
	case "structs.Infusion.address":
		x.Address = ""
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: structs.Infusion"))
		}
		panic(fmt.Errorf("message structs.Infusion does not contain field %s", fd.FullName()))
	}
}

// Get retrieves the value for a field.
//
// For unpopulated scalars, it returns the default value, where
// the default value of a bytes scalar is guaranteed to be a copy.
// For unpopulated composite types, it returns an empty, read-only view
// of the value; to obtain a mutable reference, use Mutable.
func (x *fastReflection_Infusion) Get(descriptor protoreflect.FieldDescriptor) protoreflect.Value {
	switch descriptor.FullName() {
	case "structs.Infusion.destinationType":
		value := x.DestinationType
		return protoreflect.ValueOfEnum((protoreflect.EnumNumber)(value))
	case "structs.Infusion.destinationId":
		value := x.DestinationId
		return protoreflect.ValueOfString(value)
	case "structs.Infusion.fuel":
		value := x.Fuel
		return protoreflect.ValueOfUint64(value)
	case "structs.Infusion.power":
		value := x.Power
		return protoreflect.ValueOfUint64(value)
	case "structs.Infusion.commission":
		value := x.Commission
		return protoreflect.ValueOfString(value)
	case "structs.Infusion.playerId":
		value := x.PlayerId
		return protoreflect.ValueOfString(value)
	case "structs.Infusion.address":
		value := x.Address
		return protoreflect.ValueOfString(value)
	default:
		if descriptor.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: structs.Infusion"))
		}
		panic(fmt.Errorf("message structs.Infusion does not contain field %s", descriptor.FullName()))
	}
}

// Set stores the value for a field.
//
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType.
// When setting a composite type, it is unspecified whether the stored value
// aliases the source's memory in any way. If the composite value is an
// empty, read-only value, then it panics.
//
// Set is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_Infusion) Set(fd protoreflect.FieldDescriptor, value protoreflect.Value) {
	switch fd.FullName() {
	case "structs.Infusion.destinationType":
		x.DestinationType = (ObjectType)(value.Enum())
	case "structs.Infusion.destinationId":
		x.DestinationId = value.Interface().(string)
	case "structs.Infusion.fuel":
		x.Fuel = value.Uint()
	case "structs.Infusion.power":
		x.Power = value.Uint()
	case "structs.Infusion.commission":
		x.Commission = value.Interface().(string)
	case "structs.Infusion.playerId":
		x.PlayerId = value.Interface().(string)
	case "structs.Infusion.address":
		x.Address = value.Interface().(string)
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: structs.Infusion"))
		}
		panic(fmt.Errorf("message structs.Infusion does not contain field %s", fd.FullName()))
	}
}

// Mutable returns a mutable reference to a composite type.
//
// If the field is unpopulated, it may allocate a composite value.
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType
// if not already stored.
// It panics if the field does not contain a composite type.
//
// Mutable is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_Infusion) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "structs.Infusion.destinationType":
		panic(fmt.Errorf("field destinationType of message structs.Infusion is not mutable"))
	case "structs.Infusion.destinationId":
		panic(fmt.Errorf("field destinationId of message structs.Infusion is not mutable"))
	case "structs.Infusion.fuel":
		panic(fmt.Errorf("field fuel of message structs.Infusion is not mutable"))
	case "structs.Infusion.power":
		panic(fmt.Errorf("field power of message structs.Infusion is not mutable"))
	case "structs.Infusion.commission":
		panic(fmt.Errorf("field commission of message structs.Infusion is not mutable"))
	case "structs.Infusion.playerId":
		panic(fmt.Errorf("field playerId of message structs.Infusion is not mutable"))
	case "structs.Infusion.address":
		panic(fmt.Errorf("field address of message structs.Infusion is not mutable"))
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: structs.Infusion"))
		}
		panic(fmt.Errorf("message structs.Infusion does not contain field %s", fd.FullName()))
	}
}

// NewField returns a new value that is assignable to the field
// for the given descriptor. For scalars, this returns the default value.
// For lists, maps, and messages, this returns a new, empty, mutable value.
func (x *fastReflection_Infusion) NewField(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "structs.Infusion.destinationType":
		return protoreflect.ValueOfEnum(0)
	case "structs.Infusion.destinationId":
		return protoreflect.ValueOfString("")
	case "structs.Infusion.fuel":
		return protoreflect.ValueOfUint64(uint64(0))
	case "structs.Infusion.power":
		return protoreflect.ValueOfUint64(uint64(0))
	case "structs.Infusion.commission":
		return protoreflect.ValueOfString("")
	case "structs.Infusion.playerId":
		return protoreflect.ValueOfString("")
	case "structs.Infusion.address":
		return protoreflect.ValueOfString("")
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: structs.Infusion"))
		}
		panic(fmt.Errorf("message structs.Infusion does not contain field %s", fd.FullName()))
	}
}

// WhichOneof reports which field within the oneof is populated,
// returning nil if none are populated.
// It panics if the oneof descriptor does not belong to this message.
func (x *fastReflection_Infusion) WhichOneof(d protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	switch d.FullName() {
	default:
		panic(fmt.Errorf("%s is not a oneof field in structs.Infusion", d.FullName()))
	}
	panic("unreachable")
}

// GetUnknown retrieves the entire list of unknown fields.
// The caller may only mutate the contents of the RawFields
// if the mutated bytes are stored back into the message with SetUnknown.
func (x *fastReflection_Infusion) GetUnknown() protoreflect.RawFields {
	return x.unknownFields
}

// SetUnknown stores an entire list of unknown fields.
// The raw fields must be syntactically valid according to the wire format.
// An implementation may panic if this is not the case.
// Once stored, the caller must not mutate the content of the RawFields.
// An empty RawFields may be passed to clear the fields.
//
// SetUnknown is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_Infusion) SetUnknown(fields protoreflect.RawFields) {
	x.unknownFields = fields
}

// IsValid reports whether the message is valid.
//
// An invalid message is an empty, read-only value.
//
// An invalid message often corresponds to a nil pointer of the concrete
// message type, but the details are implementation dependent.
// Validity is not part of the protobuf data model, and may not
// be preserved in marshaling or other operations.
func (x *fastReflection_Infusion) IsValid() bool {
	return x != nil
}

// ProtoMethods returns optional fastReflectionFeature-path implementations of various operations.
// This method may return nil.
//
// The returned methods type is identical to
// "google.golang.org/protobuf/runtime/protoiface".Methods.
// Consult the protoiface package documentation for details.
func (x *fastReflection_Infusion) ProtoMethods() *protoiface.Methods {
	size := func(input protoiface.SizeInput) protoiface.SizeOutput {
		x := input.Message.Interface().(*Infusion)
		if x == nil {
			return protoiface.SizeOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Size:              0,
			}
		}
		options := runtime.SizeInputToOptions(input)
		_ = options
		var n int
		var l int
		_ = l
		if x.DestinationType != 0 {
			n += 1 + runtime.Sov(uint64(x.DestinationType))
		}
		l = len(x.DestinationId)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.Fuel != 0 {
			n += 1 + runtime.Sov(uint64(x.Fuel))
		}
		if x.Power != 0 {
			n += 1 + runtime.Sov(uint64(x.Power))
		}
		l = len(x.Commission)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.PlayerId)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.Address)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.unknownFields != nil {
			n += len(x.unknownFields)
		}
		return protoiface.SizeOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Size:              n,
		}
	}

	marshal := func(input protoiface.MarshalInput) (protoiface.MarshalOutput, error) {
		x := input.Message.Interface().(*Infusion)
		if x == nil {
			return protoiface.MarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Buf:               input.Buf,
			}, nil
		}
		options := runtime.MarshalInputToOptions(input)
		_ = options
		size := options.Size(x)
		dAtA := make([]byte, size)
		i := len(dAtA)
		_ = i
		var l int
		_ = l
		if x.unknownFields != nil {
			i -= len(x.unknownFields)
			copy(dAtA[i:], x.unknownFields)
		}
		if len(x.Address) > 0 {
			i -= len(x.Address)
			copy(dAtA[i:], x.Address)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.Address)))
			i--
			dAtA[i] = 0x3a
		}
		if len(x.PlayerId) > 0 {
			i -= len(x.PlayerId)
			copy(dAtA[i:], x.PlayerId)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.PlayerId)))
			i--
			dAtA[i] = 0x32
		}
		if len(x.Commission) > 0 {
			i -= len(x.Commission)
			copy(dAtA[i:], x.Commission)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.Commission)))
			i--
			dAtA[i] = 0x2a
		}
		if x.Power != 0 {
			i = runtime.EncodeVarint(dAtA, i, uint64(x.Power))
			i--
			dAtA[i] = 0x20
		}
		if x.Fuel != 0 {
			i = runtime.EncodeVarint(dAtA, i, uint64(x.Fuel))
			i--
			dAtA[i] = 0x18
		}
		if len(x.DestinationId) > 0 {
			i -= len(x.DestinationId)
			copy(dAtA[i:], x.DestinationId)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.DestinationId)))
			i--
			dAtA[i] = 0x12
		}
		if x.DestinationType != 0 {
			i = runtime.EncodeVarint(dAtA, i, uint64(x.DestinationType))
			i--
			dAtA[i] = 0x8
		}
		if input.Buf != nil {
			input.Buf = append(input.Buf, dAtA...)
		} else {
			input.Buf = dAtA
		}
		return protoiface.MarshalOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Buf:               input.Buf,
		}, nil
	}
	unmarshal := func(input protoiface.UnmarshalInput) (protoiface.UnmarshalOutput, error) {
		x := input.Message.Interface().(*Infusion)
		if x == nil {
			return protoiface.UnmarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Flags:             input.Flags,
			}, nil
		}
		options := runtime.UnmarshalInputToOptions(input)
		_ = options
		dAtA := input.Buf
		l := len(dAtA)
		iNdEx := 0
		for iNdEx < l {
			preIndex := iNdEx
			var wire uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
				}
				if iNdEx >= l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				wire |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			fieldNum := int32(wire >> 3)
			wireType := int(wire & 0x7)
			if wireType == 4 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: Infusion: wiretype end group for non-group")
			}
			if fieldNum <= 0 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: Infusion: illegal tag %d (wire type %d)", fieldNum, wire)
			}
			switch fieldNum {
			case 1:
				if wireType != 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field DestinationType", wireType)
				}
				x.DestinationType = 0
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					x.DestinationType |= ObjectType(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
			case 2:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field DestinationId", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.DestinationId = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 3:
				if wireType != 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Fuel", wireType)
				}
				x.Fuel = 0
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					x.Fuel |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
			case 4:
				if wireType != 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Power", wireType)
				}
				x.Power = 0
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					x.Power |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
			case 5:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Commission", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.Commission = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 6:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field PlayerId", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.PlayerId = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 7:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.Address = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			default:
				iNdEx = preIndex
				skippy, err := runtime.Skip(dAtA[iNdEx:])
				if err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				if (skippy < 0) || (iNdEx+skippy) < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if (iNdEx + skippy) > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if !options.DiscardUnknown {
					x.unknownFields = append(x.unknownFields, dAtA[iNdEx:iNdEx+skippy]...)
				}
				iNdEx += skippy
			}
		}

		if iNdEx > l {
			return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
		}
		return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, nil
	}
	return &protoiface.Methods{
		NoUnkeyedLiterals: struct{}{},
		Flags:             protoiface.SupportMarshalDeterministic | protoiface.SupportUnmarshalDiscardUnknown,
		Size:              size,
		Marshal:           marshal,
		Unmarshal:         unmarshal,
		Merge:             nil,
		CheckInitialized:  nil,
	}
}

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.0
// 	protoc        (unknown)
// source: structs/structs/infusion.proto

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Infusion struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DestinationType ObjectType `protobuf:"varint,1,opt,name=destinationType,proto3,enum=structs.ObjectType" json:"destinationType,omitempty"`
	DestinationId   string     `protobuf:"bytes,2,opt,name=destinationId,proto3" json:"destinationId,omitempty"`
	Fuel            uint64     `protobuf:"varint,3,opt,name=fuel,proto3" json:"fuel,omitempty"`
	Power           uint64     `protobuf:"varint,4,opt,name=power,proto3" json:"power,omitempty"`
	Commission      string     `protobuf:"bytes,5,opt,name=commission,proto3" json:"commission,omitempty"`
	PlayerId        string     `protobuf:"bytes,6,opt,name=playerId,proto3" json:"playerId,omitempty"`
	Address         string     `protobuf:"bytes,7,opt,name=address,proto3" json:"address,omitempty"`
}

func (x *Infusion) Reset() {
	*x = Infusion{}
	if protoimpl.UnsafeEnabled {
		mi := &file_structs_structs_infusion_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Infusion) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Infusion) ProtoMessage() {}

// Deprecated: Use Infusion.ProtoReflect.Descriptor instead.
func (*Infusion) Descriptor() ([]byte, []int) {
	return file_structs_structs_infusion_proto_rawDescGZIP(), []int{0}
}

func (x *Infusion) GetDestinationType() ObjectType {
	if x != nil {
		return x.DestinationType
	}
	return ObjectType_guild
}

func (x *Infusion) GetDestinationId() string {
	if x != nil {
		return x.DestinationId
	}
	return ""
}

func (x *Infusion) GetFuel() uint64 {
	if x != nil {
		return x.Fuel
	}
	return 0
}

func (x *Infusion) GetPower() uint64 {
	if x != nil {
		return x.Power
	}
	return 0
}

func (x *Infusion) GetCommission() string {
	if x != nil {
		return x.Commission
	}
	return ""
}

func (x *Infusion) GetPlayerId() string {
	if x != nil {
		return x.PlayerId
	}
	return ""
}

func (x *Infusion) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

var File_structs_structs_infusion_proto protoreflect.FileDescriptor

var file_structs_structs_infusion_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74,
	0x73, 0x2f, 0x69, 0x6e, 0x66, 0x75, 0x73, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x07, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x1a, 0x19, 0x63, 0x6f, 0x73, 0x6d, 0x6f,
	0x73, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x14, 0x67, 0x6f, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f,
	0x67, 0x6f, 0x67, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1a, 0x73, 0x74, 0x72, 0x75,
	0x63, 0x74, 0x73, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x2f, 0x6b, 0x65, 0x79, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xa2, 0x02, 0x0a, 0x08, 0x49, 0x6e, 0x66, 0x75, 0x73,
	0x69, 0x6f, 0x6e, 0x12, 0x3d, 0x0a, 0x0f, 0x64, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x13, 0x2e, 0x73,
	0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x2e, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x54, 0x79, 0x70,
	0x65, 0x52, 0x0f, 0x64, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x24, 0x0a, 0x0d, 0x64, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x64, 0x65, 0x73, 0x74, 0x69,
	0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x66, 0x75, 0x65, 0x6c,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x04, 0x66, 0x75, 0x65, 0x6c, 0x12, 0x14, 0x0a, 0x05,
	0x70, 0x6f, 0x77, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x70, 0x6f, 0x77,
	0x65, 0x72, 0x12, 0x51, 0x0a, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x42, 0x31, 0xc8, 0xde, 0x1f, 0x00, 0xda, 0xde, 0x1f, 0x1b,
	0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x73, 0x64, 0x6b, 0x2e, 0x69, 0x6f, 0x2f, 0x6d, 0x61, 0x74,
	0x68, 0x2e, 0x4c, 0x65, 0x67, 0x61, 0x63, 0x79, 0x44, 0x65, 0x63, 0xd2, 0xb4, 0x2d, 0x0a, 0x63,
	0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x44, 0x65, 0x63, 0x52, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x69,
	0x73, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49,
	0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49,
	0x64, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x42, 0x7a, 0x0a, 0x0b, 0x63,
	0x6f, 0x6d, 0x2e, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x42, 0x0d, 0x49, 0x6e, 0x66, 0x75,
	0x73, 0x69, 0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x20, 0x63, 0x6f, 0x73,
	0x6d, 0x6f, 0x73, 0x73, 0x64, 0x6b, 0x2e, 0x69, 0x6f, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x74,
	0x72, 0x75, 0x63, 0x74, 0x73, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0xa2, 0x02, 0x03,
	0x53, 0x58, 0x58, 0xaa, 0x02, 0x07, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0xca, 0x02, 0x07,
	0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0xe2, 0x02, 0x13, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
	0x73, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x07,
	0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_structs_structs_infusion_proto_rawDescOnce sync.Once
	file_structs_structs_infusion_proto_rawDescData = file_structs_structs_infusion_proto_rawDesc
)

func file_structs_structs_infusion_proto_rawDescGZIP() []byte {
	file_structs_structs_infusion_proto_rawDescOnce.Do(func() {
		file_structs_structs_infusion_proto_rawDescData = protoimpl.X.CompressGZIP(file_structs_structs_infusion_proto_rawDescData)
	})
	return file_structs_structs_infusion_proto_rawDescData
}

var file_structs_structs_infusion_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_structs_structs_infusion_proto_goTypes = []interface{}{
	(*Infusion)(nil), // 0: structs.Infusion
	(ObjectType)(0),  // 1: structs.objectType
}
var file_structs_structs_infusion_proto_depIdxs = []int32{
	1, // 0: structs.Infusion.destinationType:type_name -> structs.objectType
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_structs_structs_infusion_proto_init() }
func file_structs_structs_infusion_proto_init() {
	if File_structs_structs_infusion_proto != nil {
		return
	}
	file_structs_structs_keys_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_structs_structs_infusion_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Infusion); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_structs_structs_infusion_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_structs_structs_infusion_proto_goTypes,
		DependencyIndexes: file_structs_structs_infusion_proto_depIdxs,
		MessageInfos:      file_structs_structs_infusion_proto_msgTypes,
	}.Build()
	File_structs_structs_infusion_proto = out.File
	file_structs_structs_infusion_proto_rawDesc = nil
	file_structs_structs_infusion_proto_goTypes = nil
	file_structs_structs_infusion_proto_depIdxs = nil
}
