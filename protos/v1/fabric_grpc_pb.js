// GENERATED CODE -- DO NOT EDIT!

// Original file comments:
// protos/v1/fabric.proto
'use strict';
var grpc = require('@grpc/grpc-js');
var v1_fabric_pb = require('../v1/fabric_pb.js');

function serialize_io_defang_v1_Event(arg) {
  if (!(arg instanceof v1_fabric_pb.Event)) {
    throw new Error('Expected argument of type io.defang.v1.Event');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Event(buffer_arg) {
  return v1_fabric_pb.Event.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_LogEntry(arg) {
  if (!(arg instanceof v1_fabric_pb.LogEntry)) {
    throw new Error('Expected argument of type io.defang.v1.LogEntry');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_LogEntry(buffer_arg) {
  return v1_fabric_pb.LogEntry.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Service(arg) {
  if (!(arg instanceof v1_fabric_pb.Service)) {
    throw new Error('Expected argument of type io.defang.v1.Service');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Service(buffer_arg) {
  return v1_fabric_pb.Service.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Services(arg) {
  if (!(arg instanceof v1_fabric_pb.Services)) {
    throw new Error('Expected argument of type io.defang.v1.Services');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Services(buffer_arg) {
  return v1_fabric_pb.Services.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Status(arg) {
  if (!(arg instanceof v1_fabric_pb.Status)) {
    throw new Error('Expected argument of type io.defang.v1.Status');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Status(buffer_arg) {
  return v1_fabric_pb.Status.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Void(arg) {
  if (!(arg instanceof v1_fabric_pb.Void)) {
    throw new Error('Expected argument of type io.defang.v1.Void');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Void(buffer_arg) {
  return v1_fabric_pb.Void.deserializeBinary(new Uint8Array(buffer_arg));
}


var FabricControllerService = exports.FabricControllerService = {
  getStatus: {
    path: '/io.defang.v1.FabricController/GetStatus',
    requestStream: false,
    responseStream: false,
    requestType: v1_fabric_pb.Void,
    responseType: v1_fabric_pb.Status,
    requestSerialize: serialize_io_defang_v1_Void,
    requestDeserialize: deserialize_io_defang_v1_Void,
    responseSerialize: serialize_io_defang_v1_Status,
    responseDeserialize: deserialize_io_defang_v1_Status,
  },
  tail: {
    path: '/io.defang.v1.FabricController/Tail',
    requestStream: false,
    responseStream: true,
    requestType: v1_fabric_pb.Void,
    responseType: v1_fabric_pb.LogEntry,
    requestSerialize: serialize_io_defang_v1_Void,
    requestDeserialize: deserialize_io_defang_v1_Void,
    responseSerialize: serialize_io_defang_v1_LogEntry,
    responseDeserialize: deserialize_io_defang_v1_LogEntry,
  },
  update: {
    path: '/io.defang.v1.FabricController/Update',
    requestStream: false,
    responseStream: false,
    requestType: v1_fabric_pb.Service,
    responseType: v1_fabric_pb.Service,
    requestSerialize: serialize_io_defang_v1_Service,
    requestDeserialize: deserialize_io_defang_v1_Service,
    responseSerialize: serialize_io_defang_v1_Service,
    responseDeserialize: deserialize_io_defang_v1_Service,
  },
  get: {
    path: '/io.defang.v1.FabricController/Get',
    requestStream: false,
    responseStream: false,
    requestType: v1_fabric_pb.Void,
    responseType: v1_fabric_pb.Service,
    requestSerialize: serialize_io_defang_v1_Void,
    requestDeserialize: deserialize_io_defang_v1_Void,
    responseSerialize: serialize_io_defang_v1_Service,
    responseDeserialize: deserialize_io_defang_v1_Service,
  },
  delete: {
    path: '/io.defang.v1.FabricController/Delete',
    requestStream: false,
    responseStream: false,
    requestType: v1_fabric_pb.Service,
    responseType: v1_fabric_pb.Void,
    requestSerialize: serialize_io_defang_v1_Service,
    requestDeserialize: deserialize_io_defang_v1_Service,
    responseSerialize: serialize_io_defang_v1_Void,
    responseDeserialize: deserialize_io_defang_v1_Void,
  },
  // rpc Promote(Void) returns (Void);
getServices: {
    path: '/io.defang.v1.FabricController/GetServices',
    requestStream: false,
    responseStream: false,
    requestType: v1_fabric_pb.Void,
    responseType: v1_fabric_pb.Services,
    requestSerialize: serialize_io_defang_v1_Void,
    requestDeserialize: deserialize_io_defang_v1_Void,
    responseSerialize: serialize_io_defang_v1_Services,
    responseDeserialize: deserialize_io_defang_v1_Services,
  },
  // only our CD service can call this
publish: {
    path: '/io.defang.v1.FabricController/Publish',
    requestStream: false,
    responseStream: false,
    requestType: v1_fabric_pb.Event,
    responseType: v1_fabric_pb.Void,
    requestSerialize: serialize_io_defang_v1_Event,
    requestDeserialize: deserialize_io_defang_v1_Event,
    responseSerialize: serialize_io_defang_v1_Void,
    responseDeserialize: deserialize_io_defang_v1_Void,
  },
};

exports.FabricControllerClient = grpc.makeGenericClientConstructor(FabricControllerService);