# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: service.proto
# Protobuf Python Version: 4.25.1
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\rservice.proto\"\x1b\n\x08Request1\x12\x0f\n\x07message\x18\x01 \x01(\t\"\x1c\n\tResponse1\x12\x0f\n\x07message\x18\x01 \x01(\t\"\x1b\n\x08Request2\x12\x0f\n\x07message\x18\x01 \x01(\t\"\x1c\n\tResponse2\x12\x0f\n\x07message\x18\x01 \x01(\t\"/\n\x11RequestInsertData\x12\x0b\n\x03key\x18\x01 \x01(\t\x12\r\n\x05value\x18\x02 \x01(\t\"%\n\x12ResopnseInsertData\x12\x0f\n\x07message\x18\x01 \x01(\t\" \n\x11RequestSelectData\x12\x0b\n\x03key\x18\x01 \x01(\t\"%\n\x12ResopnseSelectData\x12\x0f\n\x07message\x18\x01 \x01(\t2\xc8\x01\n\x0c\x44\x62Operations\x12\"\n\x07Method1\x12\t.Request1\x1a\n.Response1\"\x00\x12\"\n\x07Method2\x12\t.Request2\x1a\n.Response2\"\x00\x12\x37\n\nInsertData\x12\x12.RequestInsertData\x1a\x13.ResopnseInsertData\"\x00\x12\x37\n\nSelectData\x12\x12.RequestSelectData\x1a\x13.ResopnseSelectData\"\x00\x62\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'service_pb2', _globals)
if _descriptor._USE_C_DESCRIPTORS == False:
  DESCRIPTOR._options = None
  _globals['_REQUEST1']._serialized_start=17
  _globals['_REQUEST1']._serialized_end=44
  _globals['_RESPONSE1']._serialized_start=46
  _globals['_RESPONSE1']._serialized_end=74
  _globals['_REQUEST2']._serialized_start=76
  _globals['_REQUEST2']._serialized_end=103
  _globals['_RESPONSE2']._serialized_start=105
  _globals['_RESPONSE2']._serialized_end=133
  _globals['_REQUESTINSERTDATA']._serialized_start=135
  _globals['_REQUESTINSERTDATA']._serialized_end=182
  _globals['_RESOPNSEINSERTDATA']._serialized_start=184
  _globals['_RESOPNSEINSERTDATA']._serialized_end=221
  _globals['_REQUESTSELECTDATA']._serialized_start=223
  _globals['_REQUESTSELECTDATA']._serialized_end=255
  _globals['_RESOPNSESELECTDATA']._serialized_start=257
  _globals['_RESOPNSESELECTDATA']._serialized_end=294
  _globals['_DBOPERATIONS']._serialized_start=297
  _globals['_DBOPERATIONS']._serialized_end=497
# @@protoc_insertion_point(module_scope)