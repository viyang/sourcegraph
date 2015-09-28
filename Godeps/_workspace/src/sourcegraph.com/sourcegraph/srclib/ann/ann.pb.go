// Code generated by protoc-gen-gogo.
// source: ann.proto
// DO NOT EDIT!

/*
Package ann is a generated protocol buffer package.

It is generated from these files:
	ann.proto

It has these top-level messages:
	Ann
*/
package ann

import proto "github.com/gogo/protobuf/proto"

// discarding unused import gogoproto "github.com/gogo/protobuf/gogoproto/gogo.pb"

import sourcegraph_com_sqs_pbtypes "sourcegraph.com/sqs/pbtypes"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal

// An Ann is a source code annotation.
//
// Annotations are unique on (Repo, CommitID, UnitType, Unit, File,
// Start, End, Type).
type Ann struct {
	// Repo is the VCS repository in which this ann exists.
	Repo string `protobuf:"bytes,1,opt,name=repo,proto3" json:"Repo,omitempty"`
	// CommitID is the ID of the VCS commit that this ann exists
	// in. The CommitID is always a full commit ID (40 hexadecimal
	// characters for git and hg), never a branch or tag name.
	CommitID string `protobuf:"bytes,2,opt,name=commit_id,proto3" json:"CommitID,omitempty"`
	// UnitType is the source unit type that the annotation exists
	// on. It is either the source unit type during whose processing
	// the annotation was detected/created. Multiple annotations may
	// exist on the same file from different source unit types if a
	// file is contained in multiple source units.
	UnitType string `protobuf:"bytes,3,opt,name=unit_type,proto3" json:"UnitType,omitempty"`
	// Unit is the name of the source unit that this ann exists in.
	Unit string `protobuf:"bytes,4,opt,name=unit,proto3" json:"Unit,omitempty"`
	// File is the filename in which this Ann exists.
	File string `protobuf:"bytes,5,opt,name=file,proto3" json:"File,omitempty"`
	// Start is the byte offset of this ann's first byte in File.
	Start uint32 `protobuf:"varint,6,opt,name=start,proto3" json:"Start"`
	// End is the byte offset of this ann's last byte in File.
	End uint32 `protobuf:"varint,7,opt,name=end,proto3" json:"End"`
	// Type is the type of the annotation. See this package's type
	// constants for a list of possible types.
	Type string `protobuf:"bytes,8,opt,name=type,proto3" json:"Type"`
	// Data contains arbitrary JSON data that is specific to this
	// annotation type (e.g., the link URL for Link annotations).
	Data sourcegraph_com_sqs_pbtypes.RawMessage `protobuf:"bytes,9,opt,name=data,proto3,customtype=sourcegraph.com/sqs/pbtypes.RawMessage" json:"Data,omitempty"`
}

func (m *Ann) Reset()         { *m = Ann{} }
func (m *Ann) String() string { return proto.CompactTextString(m) }
func (*Ann) ProtoMessage()    {}

func init() {
}
