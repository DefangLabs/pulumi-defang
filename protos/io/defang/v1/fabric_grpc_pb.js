// GENERATED CODE -- DO NOT EDIT!

// Original file comments:
// protos/v1/fabric.proto
'use strict';
var grpc = require('@grpc/grpc-js');
var io_defang_v1_fabric_pb = require('../../../io/defang/v1/fabric_pb.js');
var google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js');
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');

function serialize_google_protobuf_Empty(arg) {
  if (!(arg instanceof google_protobuf_empty_pb.Empty)) {
    throw new Error('Expected argument of type google.protobuf.Empty');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_google_protobuf_Empty(buffer_arg) {
  return google_protobuf_empty_pb.Empty.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DebugRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.DebugRequest)) {
    throw new Error('Expected argument of type io.defang.v1.DebugRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DebugRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DebugRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DebugResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.DebugResponse)) {
    throw new Error('Expected argument of type io.defang.v1.DebugResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DebugResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.DebugResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DelegateSubdomainZoneRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.DelegateSubdomainZoneRequest)) {
    throw new Error('Expected argument of type io.defang.v1.DelegateSubdomainZoneRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DelegateSubdomainZoneRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DelegateSubdomainZoneRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DelegateSubdomainZoneResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse)) {
    throw new Error('Expected argument of type io.defang.v1.DelegateSubdomainZoneResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DelegateSubdomainZoneResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DeleteRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.DeleteRequest)) {
    throw new Error('Expected argument of type io.defang.v1.DeleteRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DeleteRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DeleteRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DeleteResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.DeleteResponse)) {
    throw new Error('Expected argument of type io.defang.v1.DeleteResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DeleteResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.DeleteResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DeployRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.DeployRequest)) {
    throw new Error('Expected argument of type io.defang.v1.DeployRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DeployRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DeployRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DeployResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.DeployResponse)) {
    throw new Error('Expected argument of type io.defang.v1.DeployResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DeployResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.DeployResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GenerateFilesRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.GenerateFilesRequest)) {
    throw new Error('Expected argument of type io.defang.v1.GenerateFilesRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GenerateFilesRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.GenerateFilesRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GenerateFilesResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.GenerateFilesResponse)) {
    throw new Error('Expected argument of type io.defang.v1.GenerateFilesResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GenerateFilesResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.GenerateFilesResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GenerateStatusRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.GenerateStatusRequest)) {
    throw new Error('Expected argument of type io.defang.v1.GenerateStatusRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GenerateStatusRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.GenerateStatusRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_ListServicesResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.ListServicesResponse)) {
    throw new Error('Expected argument of type io.defang.v1.ListServicesResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_ListServicesResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.ListServicesResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_PublishRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.PublishRequest)) {
    throw new Error('Expected argument of type io.defang.v1.PublishRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_PublishRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.PublishRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_SecretValue(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.SecretValue)) {
    throw new Error('Expected argument of type io.defang.v1.SecretValue');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_SecretValue(buffer_arg) {
  return io_defang_v1_fabric_pb.SecretValue.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Secrets(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.Secrets)) {
    throw new Error('Expected argument of type io.defang.v1.Secrets');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Secrets(buffer_arg) {
  return io_defang_v1_fabric_pb.Secrets.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Service(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.Service)) {
    throw new Error('Expected argument of type io.defang.v1.Service');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Service(buffer_arg) {
  return io_defang_v1_fabric_pb.Service.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_ServiceID(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.ServiceID)) {
    throw new Error('Expected argument of type io.defang.v1.ServiceID');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_ServiceID(buffer_arg) {
  return io_defang_v1_fabric_pb.ServiceID.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_ServiceInfo(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.ServiceInfo)) {
    throw new Error('Expected argument of type io.defang.v1.ServiceInfo');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_ServiceInfo(buffer_arg) {
  return io_defang_v1_fabric_pb.ServiceInfo.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_StartGenerateResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.StartGenerateResponse)) {
    throw new Error('Expected argument of type io.defang.v1.StartGenerateResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_StartGenerateResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.StartGenerateResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Status(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.Status)) {
    throw new Error('Expected argument of type io.defang.v1.Status');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Status(buffer_arg) {
  return io_defang_v1_fabric_pb.Status.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_SubscribeRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.SubscribeRequest)) {
    throw new Error('Expected argument of type io.defang.v1.SubscribeRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_SubscribeRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.SubscribeRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_SubscribeResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.SubscribeResponse)) {
    throw new Error('Expected argument of type io.defang.v1.SubscribeResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_SubscribeResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.SubscribeResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TailRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.TailRequest)) {
    throw new Error('Expected argument of type io.defang.v1.TailRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TailRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.TailRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TailResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.TailResponse)) {
    throw new Error('Expected argument of type io.defang.v1.TailResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TailResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.TailResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TokenRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.TokenRequest)) {
    throw new Error('Expected argument of type io.defang.v1.TokenRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TokenRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.TokenRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TokenResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.TokenResponse)) {
    throw new Error('Expected argument of type io.defang.v1.TokenResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TokenResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.TokenResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TrackRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.TrackRequest)) {
    throw new Error('Expected argument of type io.defang.v1.TrackRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TrackRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.TrackRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_UploadURLRequest(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.UploadURLRequest)) {
    throw new Error('Expected argument of type io.defang.v1.UploadURLRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_UploadURLRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.UploadURLRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_UploadURLResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.UploadURLResponse)) {
    throw new Error('Expected argument of type io.defang.v1.UploadURLResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_UploadURLResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.UploadURLResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Version(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.Version)) {
    throw new Error('Expected argument of type io.defang.v1.Version');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Version(buffer_arg) {
  return io_defang_v1_fabric_pb.Version.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_WhoAmIResponse(arg) {
  if (!(arg instanceof io_defang_v1_fabric_pb.WhoAmIResponse)) {
    throw new Error('Expected argument of type io.defang.v1.WhoAmIResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_WhoAmIResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.WhoAmIResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var FabricControllerService = exports.FabricControllerService = {
  getStatus: {
    path: '/io.defang.v1.FabricController/GetStatus',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: io_defang_v1_fabric_pb.Status,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_io_defang_v1_Status,
    responseDeserialize: deserialize_io_defang_v1_Status,
  },
  getVersion: {
    path: '/io.defang.v1.FabricController/GetVersion',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: io_defang_v1_fabric_pb.Version,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_io_defang_v1_Version,
    responseDeserialize: deserialize_io_defang_v1_Version,
  },
  token: {
    path: '/io.defang.v1.FabricController/Token',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.TokenRequest,
    responseType: io_defang_v1_fabric_pb.TokenResponse,
    requestSerialize: serialize_io_defang_v1_TokenRequest,
    requestDeserialize: deserialize_io_defang_v1_TokenRequest,
    responseSerialize: serialize_io_defang_v1_TokenResponse,
    responseDeserialize: deserialize_io_defang_v1_TokenResponse,
  },
  // public
revokeToken: {
    path: '/io.defang.v1.FabricController/RevokeToken',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  tail: {
    path: '/io.defang.v1.FabricController/Tail',
    requestStream: false,
    responseStream: true,
    requestType: io_defang_v1_fabric_pb.TailRequest,
    responseType: io_defang_v1_fabric_pb.TailResponse,
    requestSerialize: serialize_io_defang_v1_TailRequest,
    requestDeserialize: deserialize_io_defang_v1_TailRequest,
    responseSerialize: serialize_io_defang_v1_TailResponse,
    responseDeserialize: deserialize_io_defang_v1_TailResponse,
  },
  update: {
    path: '/io.defang.v1.FabricController/Update',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.Service,
    responseType: io_defang_v1_fabric_pb.ServiceInfo,
    requestSerialize: serialize_io_defang_v1_Service,
    requestDeserialize: deserialize_io_defang_v1_Service,
    responseSerialize: serialize_io_defang_v1_ServiceInfo,
    responseDeserialize: deserialize_io_defang_v1_ServiceInfo,
  },
  // deprecated; use Deploy
deploy: {
    path: '/io.defang.v1.FabricController/Deploy',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.DeployRequest,
    responseType: io_defang_v1_fabric_pb.DeployResponse,
    requestSerialize: serialize_io_defang_v1_DeployRequest,
    requestDeserialize: deserialize_io_defang_v1_DeployRequest,
    responseSerialize: serialize_io_defang_v1_DeployResponse,
    responseDeserialize: deserialize_io_defang_v1_DeployResponse,
  },
  get: {
    path: '/io.defang.v1.FabricController/Get',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.ServiceID,
    responseType: io_defang_v1_fabric_pb.ServiceInfo,
    requestSerialize: serialize_io_defang_v1_ServiceID,
    requestDeserialize: deserialize_io_defang_v1_ServiceID,
    responseSerialize: serialize_io_defang_v1_ServiceInfo,
    responseDeserialize: deserialize_io_defang_v1_ServiceInfo,
  },
  delete: {
    path: '/io.defang.v1.FabricController/Delete',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.DeleteRequest,
    responseType: io_defang_v1_fabric_pb.DeleteResponse,
    requestSerialize: serialize_io_defang_v1_DeleteRequest,
    requestDeserialize: deserialize_io_defang_v1_DeleteRequest,
    responseSerialize: serialize_io_defang_v1_DeleteResponse,
    responseDeserialize: deserialize_io_defang_v1_DeleteResponse,
  },
  // deprecated; use Deploy
publish: {
    path: '/io.defang.v1.FabricController/Publish',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.PublishRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_PublishRequest,
    requestDeserialize: deserialize_io_defang_v1_PublishRequest,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  subscribe: {
    path: '/io.defang.v1.FabricController/Subscribe',
    requestStream: false,
    responseStream: true,
    requestType: io_defang_v1_fabric_pb.SubscribeRequest,
    responseType: io_defang_v1_fabric_pb.SubscribeResponse,
    requestSerialize: serialize_io_defang_v1_SubscribeRequest,
    requestDeserialize: deserialize_io_defang_v1_SubscribeRequest,
    responseSerialize: serialize_io_defang_v1_SubscribeResponse,
    responseDeserialize: deserialize_io_defang_v1_SubscribeResponse,
  },
  // rpc Promote(google.protobuf.Empty) returns (google.protobuf.Empty);
getServices: {
    path: '/io.defang.v1.FabricController/GetServices',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: io_defang_v1_fabric_pb.ListServicesResponse,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_io_defang_v1_ListServicesResponse,
    responseDeserialize: deserialize_io_defang_v1_ListServicesResponse,
  },
  generateFiles: {
    path: '/io.defang.v1.FabricController/GenerateFiles',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.GenerateFilesRequest,
    responseType: io_defang_v1_fabric_pb.GenerateFilesResponse,
    requestSerialize: serialize_io_defang_v1_GenerateFilesRequest,
    requestDeserialize: deserialize_io_defang_v1_GenerateFilesRequest,
    responseSerialize: serialize_io_defang_v1_GenerateFilesResponse,
    responseDeserialize: deserialize_io_defang_v1_GenerateFilesResponse,
  },
  // deprecated; use StartGenerate/GenerateStatus
startGenerate: {
    path: '/io.defang.v1.FabricController/StartGenerate',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.GenerateFilesRequest,
    responseType: io_defang_v1_fabric_pb.StartGenerateResponse,
    requestSerialize: serialize_io_defang_v1_GenerateFilesRequest,
    requestDeserialize: deserialize_io_defang_v1_GenerateFilesRequest,
    responseSerialize: serialize_io_defang_v1_StartGenerateResponse,
    responseDeserialize: deserialize_io_defang_v1_StartGenerateResponse,
  },
  generateStatus: {
    path: '/io.defang.v1.FabricController/GenerateStatus',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.GenerateStatusRequest,
    responseType: io_defang_v1_fabric_pb.GenerateFilesResponse,
    requestSerialize: serialize_io_defang_v1_GenerateStatusRequest,
    requestDeserialize: deserialize_io_defang_v1_GenerateStatusRequest,
    responseSerialize: serialize_io_defang_v1_GenerateFilesResponse,
    responseDeserialize: deserialize_io_defang_v1_GenerateFilesResponse,
  },
  debug: {
    path: '/io.defang.v1.FabricController/Debug',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.DebugRequest,
    responseType: io_defang_v1_fabric_pb.DebugResponse,
    requestSerialize: serialize_io_defang_v1_DebugRequest,
    requestDeserialize: deserialize_io_defang_v1_DebugRequest,
    responseSerialize: serialize_io_defang_v1_DebugResponse,
    responseDeserialize: deserialize_io_defang_v1_DebugResponse,
  },
  // TEST
signEULA: {
    path: '/io.defang.v1.FabricController/SignEULA',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  // AgreeToS
checkToS: {
    path: '/io.defang.v1.FabricController/CheckToS',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  putSecret: {
    path: '/io.defang.v1.FabricController/PutSecret',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.SecretValue,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_SecretValue,
    requestDeserialize: deserialize_io_defang_v1_SecretValue,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  deleteSecrets: {
    path: '/io.defang.v1.FabricController/DeleteSecrets',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.Secrets,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_Secrets,
    requestDeserialize: deserialize_io_defang_v1_Secrets,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  listSecrets: {
    path: '/io.defang.v1.FabricController/ListSecrets',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: io_defang_v1_fabric_pb.Secrets,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_io_defang_v1_Secrets,
    responseDeserialize: deserialize_io_defang_v1_Secrets,
  },
  createUploadURL: {
    path: '/io.defang.v1.FabricController/CreateUploadURL',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.UploadURLRequest,
    responseType: io_defang_v1_fabric_pb.UploadURLResponse,
    requestSerialize: serialize_io_defang_v1_UploadURLRequest,
    requestDeserialize: deserialize_io_defang_v1_UploadURLRequest,
    responseSerialize: serialize_io_defang_v1_UploadURLResponse,
    responseDeserialize: deserialize_io_defang_v1_UploadURLResponse,
  },
  delegateSubdomainZone: {
    path: '/io.defang.v1.FabricController/DelegateSubdomainZone',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.DelegateSubdomainZoneRequest,
    responseType: io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse,
    requestSerialize: serialize_io_defang_v1_DelegateSubdomainZoneRequest,
    requestDeserialize: deserialize_io_defang_v1_DelegateSubdomainZoneRequest,
    responseSerialize: serialize_io_defang_v1_DelegateSubdomainZoneResponse,
    responseDeserialize: deserialize_io_defang_v1_DelegateSubdomainZoneResponse,
  },
  deleteSubdomainZone: {
    path: '/io.defang.v1.FabricController/DeleteSubdomainZone',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  getDelegateSubdomainZone: {
    path: '/io.defang.v1.FabricController/GetDelegateSubdomainZone',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_io_defang_v1_DelegateSubdomainZoneResponse,
    responseDeserialize: deserialize_io_defang_v1_DelegateSubdomainZoneResponse,
  },
  whoAmI: {
    path: '/io.defang.v1.FabricController/WhoAmI',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: io_defang_v1_fabric_pb.WhoAmIResponse,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_io_defang_v1_WhoAmIResponse,
    responseDeserialize: deserialize_io_defang_v1_WhoAmIResponse,
  },
  track: {
    path: '/io.defang.v1.FabricController/Track',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.TrackRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_TrackRequest,
    requestDeserialize: deserialize_io_defang_v1_TrackRequest,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
};

exports.FabricControllerClient = grpc.makeGenericClientConstructor(FabricControllerService);
