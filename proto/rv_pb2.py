# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: rv.proto
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x08rv.proto\x12\x08rv.proto\"\xde\x01\n\x0b\x46ileRequest\x12\x10\n\x08\x66ilename\x18\x01 \x01(\t\x12\x0e\n\x06md5sum\x18\x02 \x01(\t\x12\x0f\n\x07\x63ontent\x18\x03 \x01(\x0c\x12\x13\n\x0b\x63onvert_sql\x18\x04 \x01(\x08\x12.\n\x07project\x18\x05 \x01(\x0e\x32\x1d.rv.proto.FileRequest.Project\"W\n\x07Project\x12\x0b\n\x07UNKNOWN\x10\x00\x12\x0e\n\nROUTEVIEWS\x10\x01\x12\x12\n\x0eROUTEVIEWS_RIB\x10\x04\x12\x0c\n\x08RIPE_RIS\x10\x02\x12\r\n\tRPKI_RARC\x10\x03\"\x82\x01\n\x0c\x46ileResponse\x12-\n\x06status\x18\x01 \x01(\x0e\x32\x1d.rv.proto.FileResponse.Status\x12\x15\n\rerror_message\x18\x02 \x01(\t\",\n\x06Status\x12\x0b\n\x07UNKNOWN\x10\x00\x12\x0b\n\x07SUCCESS\x10\x01\x12\x08\n\x04\x46\x41IL\x10\x02\x32\x41\n\x02RV\x12;\n\nFileUpload\x12\x15.rv.proto.FileRequest\x1a\x16.rv.proto.FileResponseB5Z3github.com/routeviews/google-cloud-storage/proto/rvb\x06proto3')



_FILEREQUEST = DESCRIPTOR.message_types_by_name['FileRequest']
_FILERESPONSE = DESCRIPTOR.message_types_by_name['FileResponse']
_FILEREQUEST_PROJECT = _FILEREQUEST.enum_types_by_name['Project']
_FILERESPONSE_STATUS = _FILERESPONSE.enum_types_by_name['Status']
FileRequest = _reflection.GeneratedProtocolMessageType('FileRequest', (_message.Message,), {
  'DESCRIPTOR' : _FILEREQUEST,
  '__module__' : 'rv_pb2'
  # @@protoc_insertion_point(class_scope:rv.proto.FileRequest)
  })
_sym_db.RegisterMessage(FileRequest)

FileResponse = _reflection.GeneratedProtocolMessageType('FileResponse', (_message.Message,), {
  'DESCRIPTOR' : _FILERESPONSE,
  '__module__' : 'rv_pb2'
  # @@protoc_insertion_point(class_scope:rv.proto.FileResponse)
  })
_sym_db.RegisterMessage(FileResponse)

_RV = DESCRIPTOR.services_by_name['RV']
if _descriptor._USE_C_DESCRIPTORS == False:

  DESCRIPTOR._options = None
  DESCRIPTOR._serialized_options = b'Z3github.com/routeviews/google-cloud-storage/proto/rv'
  _FILEREQUEST._serialized_start=23
  _FILEREQUEST._serialized_end=245
  _FILEREQUEST_PROJECT._serialized_start=158
  _FILEREQUEST_PROJECT._serialized_end=245
  _FILERESPONSE._serialized_start=248
  _FILERESPONSE._serialized_end=378
  _FILERESPONSE_STATUS._serialized_start=334
  _FILERESPONSE_STATUS._serialized_end=378
  _RV._serialized_start=380
  _RV._serialized_end=445
# @@protoc_insertion_point(module_scope)