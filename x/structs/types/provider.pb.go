// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/provider.proto

package types

import (
	cosmossdk_io_math "cosmossdk.io/math"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type Provider struct {
	Id                          string                                   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Index                       uint64                                   `protobuf:"varint,2,opt,name=index,proto3" json:"index,omitempty"`
	SubstationId                string                                   `protobuf:"bytes,3,opt,name=substationId,proto3" json:"substationId,omitempty"`
	Rate                        github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,4,rep,name=rate,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"rate"`
	AccessPolicy                ProviderAccessPolicy                     `protobuf:"varint,5,opt,name=accessPolicy,proto3,enum=structs.structs.ProviderAccessPolicy" json:"accessPolicy,omitempty"`
	MinimumCapacity             uint64                                   `protobuf:"varint,6,opt,name=minimumCapacity,proto3" json:"minimumCapacity,omitempty"`
	MaximumCapacity             uint64                                   `protobuf:"varint,7,opt,name=maximumCapacity,proto3" json:"maximumCapacity,omitempty"`
	MinimumDuration             uint64                                   `protobuf:"varint,8,opt,name=minimumDuration,proto3" json:"minimumDuration,omitempty"`
	MaximumDuration             uint64                                   `protobuf:"varint,9,opt,name=maximumDuration,proto3" json:"maximumDuration,omitempty"`
	ProviderCancellationPenalty cosmossdk_io_math.LegacyDec              `protobuf:"bytes,10,opt,name=providerCancellationPenalty,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"providerCancellationPenalty"`
	ConsumerCancellationPenalty cosmossdk_io_math.LegacyDec              `protobuf:"bytes,11,opt,name=consumerCancellationPenalty,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"consumerCancellationPenalty"`
	PayoutAddress               string                                   `protobuf:"bytes,12,opt,name=payoutAddress,proto3" json:"payoutAddress,omitempty"`
	Creator                     string                                   `protobuf:"bytes,13,opt,name=creator,proto3" json:"creator,omitempty"`
	Owner                       string                                   `protobuf:"bytes,14,opt,name=owner,proto3" json:"owner,omitempty"`
}

func (m *Provider) Reset()         { *m = Provider{} }
func (m *Provider) String() string { return proto.CompactTextString(m) }
func (*Provider) ProtoMessage()    {}
func (*Provider) Descriptor() ([]byte, []int) {
	return fileDescriptor_8276b5888e8b3af8, []int{0}
}
func (m *Provider) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Provider) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Provider.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Provider) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Provider.Merge(m, src)
}
func (m *Provider) XXX_Size() int {
	return m.Size()
}
func (m *Provider) XXX_DiscardUnknown() {
	xxx_messageInfo_Provider.DiscardUnknown(m)
}

var xxx_messageInfo_Provider proto.InternalMessageInfo

func (m *Provider) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Provider) GetIndex() uint64 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *Provider) GetSubstationId() string {
	if m != nil {
		return m.SubstationId
	}
	return ""
}

func (m *Provider) GetRate() github_com_cosmos_cosmos_sdk_types.Coins {
	if m != nil {
		return m.Rate
	}
	return nil
}

func (m *Provider) GetAccessPolicy() ProviderAccessPolicy {
	if m != nil {
		return m.AccessPolicy
	}
	return ProviderAccessPolicy_openMarket
}

func (m *Provider) GetMinimumCapacity() uint64 {
	if m != nil {
		return m.MinimumCapacity
	}
	return 0
}

func (m *Provider) GetMaximumCapacity() uint64 {
	if m != nil {
		return m.MaximumCapacity
	}
	return 0
}

func (m *Provider) GetMinimumDuration() uint64 {
	if m != nil {
		return m.MinimumDuration
	}
	return 0
}

func (m *Provider) GetMaximumDuration() uint64 {
	if m != nil {
		return m.MaximumDuration
	}
	return 0
}

func (m *Provider) GetPayoutAddress() string {
	if m != nil {
		return m.PayoutAddress
	}
	return ""
}

func (m *Provider) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *Provider) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

func init() {
	proto.RegisterType((*Provider)(nil), "structs.structs.Provider")
}

func init() { proto.RegisterFile("structs/structs/provider.proto", fileDescriptor_8276b5888e8b3af8) }

var fileDescriptor_8276b5888e8b3af8 = []byte{
	// 524 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x53, 0x3d, 0x6b, 0x1b, 0x3f,
	0x1c, 0xf6, 0x39, 0x76, 0x5e, 0x14, 0xc7, 0xe1, 0x7f, 0x04, 0xfe, 0x8a, 0x03, 0x67, 0x13, 0x5a,
	0x38, 0x0a, 0xd1, 0xe1, 0x94, 0x2e, 0xdd, 0x62, 0x7b, 0x09, 0x74, 0x30, 0x1e, 0xbb, 0x14, 0x59,
	0x12, 0x8e, 0xb0, 0x4f, 0x32, 0x92, 0x2e, 0xf5, 0x7d, 0x8b, 0x7e, 0x84, 0x8e, 0xa5, 0x53, 0x87,
	0x7e, 0x88, 0x8c, 0xa1, 0x53, 0xe9, 0x90, 0x16, 0x7b, 0x68, 0xf7, 0x7e, 0x81, 0x72, 0x3a, 0x9d,
	0xf1, 0x99, 0x92, 0xa1, 0xcb, 0xe9, 0x7e, 0x8f, 0x1e, 0x3d, 0xbf, 0x77, 0x10, 0x68, 0xa3, 0x12,
	0x62, 0x74, 0x54, 0x9c, 0x73, 0x25, 0x6f, 0x39, 0x65, 0x0a, 0xcd, 0x95, 0x34, 0xd2, 0x3f, 0x76,
	0x38, 0x72, 0x67, 0xeb, 0x94, 0x48, 0x1d, 0x4b, 0xfd, 0xc6, 0x5e, 0x47, 0xb9, 0x91, 0x73, 0x5b,
	0x27, 0x13, 0x39, 0x91, 0x39, 0x9e, 0xfd, 0x39, 0x34, 0xc8, 0x39, 0xd1, 0x18, 0x6b, 0x16, 0xdd,
	0x76, 0xc7, 0xcc, 0xe0, 0x6e, 0x44, 0x24, 0x17, 0xee, 0xfe, 0x3f, 0x1c, 0x73, 0x21, 0x23, 0xfb,
	0x75, 0x50, 0x6b, 0x3b, 0xa8, 0x29, 0x4b, 0x9d, 0x93, 0xf3, 0xdf, 0x75, 0xb0, 0x3f, 0x74, 0x31,
	0xfa, 0x4d, 0x50, 0xe5, 0x14, 0x7a, 0x1d, 0x2f, 0x3c, 0x18, 0x55, 0x39, 0xf5, 0x4f, 0x40, 0x9d,
	0x0b, 0xca, 0x16, 0xb0, 0xda, 0xf1, 0xc2, 0xda, 0x28, 0x37, 0xfc, 0x73, 0xd0, 0xd0, 0xc9, 0x58,
	0x1b, 0x6c, 0xb8, 0x14, 0xd7, 0x14, 0xee, 0x58, 0x7e, 0x09, 0xf3, 0x29, 0xa8, 0x29, 0x6c, 0x18,
	0xac, 0x75, 0x76, 0xc2, 0xc3, 0xcb, 0x53, 0xe4, 0x12, 0xcb, 0x82, 0x46, 0x2e, 0x68, 0xd4, 0x97,
	0x5c, 0xf4, 0x5e, 0xdc, 0x3d, 0xb4, 0x2b, 0x1f, 0xbf, 0xb7, 0xc3, 0x09, 0x37, 0x37, 0xc9, 0x18,
	0x11, 0x19, 0xbb, 0x2a, 0xb8, 0xe3, 0x42, 0xd3, 0x69, 0x64, 0xd2, 0x39, 0xd3, 0xf6, 0x81, 0xfe,
	0xf0, 0xf3, 0xd3, 0x33, 0x6f, 0x64, 0xd5, 0xfd, 0x6b, 0xd0, 0xc0, 0x84, 0x30, 0xad, 0x87, 0x72,
	0xc6, 0x49, 0x0a, 0xeb, 0x1d, 0x2f, 0x6c, 0x5e, 0x3e, 0x45, 0x5b, 0x45, 0x46, 0x45, 0x13, 0xae,
	0x36, 0xc8, 0xa3, 0xd2, 0x53, 0x3f, 0x04, 0xc7, 0x31, 0x17, 0x3c, 0x4e, 0xe2, 0x3e, 0x9e, 0x63,
	0xc2, 0x4d, 0x0a, 0x77, 0x6d, 0xd2, 0xdb, 0xb0, 0x65, 0xe2, 0x45, 0x89, 0xb9, 0xe7, 0x98, 0x65,
	0x78, 0x43, 0x73, 0x90, 0x28, 0x5b, 0x19, 0xb8, 0x5f, 0xd2, 0x2c, 0xe0, 0x0d, 0xcd, 0x35, 0xf3,
	0xa0, 0xa4, 0xb9, 0x66, 0x6a, 0x70, 0x56, 0x64, 0xd3, 0xc7, 0x82, 0xb0, 0xd9, 0xcc, 0xe2, 0x43,
	0x26, 0xf0, 0xcc, 0xa4, 0x10, 0x64, 0xbd, 0xe8, 0x75, 0xb3, 0xa2, 0x7e, 0x7b, 0x68, 0x9f, 0xe5,
	0x25, 0xd4, 0x74, 0x8a, 0xb8, 0x8c, 0x62, 0x6c, 0x6e, 0xd0, 0x2b, 0x36, 0xc1, 0x24, 0x1d, 0x30,
	0xf2, 0xe5, 0xf3, 0x05, 0x70, 0x5d, 0x19, 0x30, 0x32, 0x7a, 0x4c, 0x35, 0x73, 0x4a, 0xa4, 0xd0,
	0x49, 0xfc, 0x77, 0xa7, 0x87, 0xff, 0xec, 0xf4, 0x11, 0x55, 0xff, 0x09, 0x38, 0x9a, 0xe3, 0x54,
	0x26, 0xe6, 0x8a, 0x52, 0xc5, 0xb4, 0x86, 0x0d, 0x3b, 0x67, 0x65, 0xd0, 0x87, 0x60, 0x8f, 0x28,
	0x86, 0x8d, 0x54, 0xf0, 0xc8, 0xde, 0x17, 0x66, 0x36, 0xbc, 0xf2, 0xad, 0x60, 0x0a, 0x36, 0x2d,
	0x9e, 0x1b, 0x2f, 0x6b, 0xbf, 0xde, 0xb7, 0xbd, 0x5e, 0xf7, 0x6e, 0x19, 0x78, 0xf7, 0xcb, 0xc0,
	0xfb, 0xb1, 0x0c, 0xbc, 0x77, 0xab, 0xa0, 0x72, 0xbf, 0x0a, 0x2a, 0x5f, 0x57, 0x41, 0xe5, 0xf5,
	0xff, 0xc5, 0x8e, 0x2c, 0xd6, 0xdb, 0x62, 0x87, 0x6f, 0xbc, 0x6b, 0xf7, 0xe5, 0xf9, 0x9f, 0x00,
	0x00, 0x00, 0xff, 0xff, 0x93, 0x47, 0x22, 0x7f, 0xe2, 0x03, 0x00, 0x00,
}

func (this *Provider) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Provider)
	if !ok {
		that2, ok := that.(Provider)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.Id != that1.Id {
		return false
	}
	if this.Index != that1.Index {
		return false
	}
	if this.SubstationId != that1.SubstationId {
		return false
	}
	if len(this.Rate) != len(that1.Rate) {
		return false
	}
	for i := range this.Rate {
		if !this.Rate[i].Equal(&that1.Rate[i]) {
			return false
		}
	}
	if this.AccessPolicy != that1.AccessPolicy {
		return false
	}
	if this.MinimumCapacity != that1.MinimumCapacity {
		return false
	}
	if this.MaximumCapacity != that1.MaximumCapacity {
		return false
	}
	if this.MinimumDuration != that1.MinimumDuration {
		return false
	}
	if this.MaximumDuration != that1.MaximumDuration {
		return false
	}
	if !this.ProviderCancellationPenalty.Equal(that1.ProviderCancellationPenalty) {
		return false
	}
	if !this.ConsumerCancellationPenalty.Equal(that1.ConsumerCancellationPenalty) {
		return false
	}
	if this.PayoutAddress != that1.PayoutAddress {
		return false
	}
	if this.Creator != that1.Creator {
		return false
	}
	if this.Owner != that1.Owner {
		return false
	}
	return true
}
func (m *Provider) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Provider) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Provider) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Owner) > 0 {
		i -= len(m.Owner)
		copy(dAtA[i:], m.Owner)
		i = encodeVarintProvider(dAtA, i, uint64(len(m.Owner)))
		i--
		dAtA[i] = 0x72
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintProvider(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x6a
	}
	if len(m.PayoutAddress) > 0 {
		i -= len(m.PayoutAddress)
		copy(dAtA[i:], m.PayoutAddress)
		i = encodeVarintProvider(dAtA, i, uint64(len(m.PayoutAddress)))
		i--
		dAtA[i] = 0x62
	}
	{
		size := m.ConsumerCancellationPenalty.Size()
		i -= size
		if _, err := m.ConsumerCancellationPenalty.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintProvider(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x5a
	{
		size := m.ProviderCancellationPenalty.Size()
		i -= size
		if _, err := m.ProviderCancellationPenalty.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintProvider(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x52
	if m.MaximumDuration != 0 {
		i = encodeVarintProvider(dAtA, i, uint64(m.MaximumDuration))
		i--
		dAtA[i] = 0x48
	}
	if m.MinimumDuration != 0 {
		i = encodeVarintProvider(dAtA, i, uint64(m.MinimumDuration))
		i--
		dAtA[i] = 0x40
	}
	if m.MaximumCapacity != 0 {
		i = encodeVarintProvider(dAtA, i, uint64(m.MaximumCapacity))
		i--
		dAtA[i] = 0x38
	}
	if m.MinimumCapacity != 0 {
		i = encodeVarintProvider(dAtA, i, uint64(m.MinimumCapacity))
		i--
		dAtA[i] = 0x30
	}
	if m.AccessPolicy != 0 {
		i = encodeVarintProvider(dAtA, i, uint64(m.AccessPolicy))
		i--
		dAtA[i] = 0x28
	}
	if len(m.Rate) > 0 {
		for iNdEx := len(m.Rate) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Rate[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintProvider(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.SubstationId) > 0 {
		i -= len(m.SubstationId)
		copy(dAtA[i:], m.SubstationId)
		i = encodeVarintProvider(dAtA, i, uint64(len(m.SubstationId)))
		i--
		dAtA[i] = 0x1a
	}
	if m.Index != 0 {
		i = encodeVarintProvider(dAtA, i, uint64(m.Index))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintProvider(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintProvider(dAtA []byte, offset int, v uint64) int {
	offset -= sovProvider(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Provider) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovProvider(uint64(l))
	}
	if m.Index != 0 {
		n += 1 + sovProvider(uint64(m.Index))
	}
	l = len(m.SubstationId)
	if l > 0 {
		n += 1 + l + sovProvider(uint64(l))
	}
	if len(m.Rate) > 0 {
		for _, e := range m.Rate {
			l = e.Size()
			n += 1 + l + sovProvider(uint64(l))
		}
	}
	if m.AccessPolicy != 0 {
		n += 1 + sovProvider(uint64(m.AccessPolicy))
	}
	if m.MinimumCapacity != 0 {
		n += 1 + sovProvider(uint64(m.MinimumCapacity))
	}
	if m.MaximumCapacity != 0 {
		n += 1 + sovProvider(uint64(m.MaximumCapacity))
	}
	if m.MinimumDuration != 0 {
		n += 1 + sovProvider(uint64(m.MinimumDuration))
	}
	if m.MaximumDuration != 0 {
		n += 1 + sovProvider(uint64(m.MaximumDuration))
	}
	l = m.ProviderCancellationPenalty.Size()
	n += 1 + l + sovProvider(uint64(l))
	l = m.ConsumerCancellationPenalty.Size()
	n += 1 + l + sovProvider(uint64(l))
	l = len(m.PayoutAddress)
	if l > 0 {
		n += 1 + l + sovProvider(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovProvider(uint64(l))
	}
	l = len(m.Owner)
	if l > 0 {
		n += 1 + l + sovProvider(uint64(l))
	}
	return n
}

func sovProvider(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozProvider(x uint64) (n int) {
	return sovProvider(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Provider) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProvider
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
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
			return fmt.Errorf("proto: Provider: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Provider: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
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
				return ErrInvalidLengthProvider
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProvider
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Index", wireType)
			}
			m.Index = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Index |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SubstationId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
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
				return ErrInvalidLengthProvider
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProvider
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SubstationId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Rate", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProvider
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthProvider
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Rate = append(m.Rate, types.Coin{})
			if err := m.Rate[len(m.Rate)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AccessPolicy", wireType)
			}
			m.AccessPolicy = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.AccessPolicy |= ProviderAccessPolicy(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinimumCapacity", wireType)
			}
			m.MinimumCapacity = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MinimumCapacity |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaximumCapacity", wireType)
			}
			m.MaximumCapacity = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaximumCapacity |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinimumDuration", wireType)
			}
			m.MinimumDuration = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MinimumDuration |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaximumDuration", wireType)
			}
			m.MaximumDuration = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaximumDuration |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProviderCancellationPenalty", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
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
				return ErrInvalidLengthProvider
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProvider
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ProviderCancellationPenalty.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ConsumerCancellationPenalty", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
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
				return ErrInvalidLengthProvider
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProvider
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ConsumerCancellationPenalty.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 12:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PayoutAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
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
				return ErrInvalidLengthProvider
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProvider
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PayoutAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 13:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
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
				return ErrInvalidLengthProvider
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProvider
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 14:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Owner", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
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
				return ErrInvalidLengthProvider
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProvider
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Owner = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProvider(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthProvider
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipProvider(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowProvider
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowProvider
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthProvider
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupProvider
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthProvider
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthProvider        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowProvider          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupProvider = fmt.Errorf("proto: unexpected end of group")
)
