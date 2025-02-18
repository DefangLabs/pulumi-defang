// GENERATED CODE -- DO NOT EDIT!

// Original file comments:
// protos/io/defang/v1/fabric.proto
'use strict';
var grpc = require('@grpc/grpc-js');
var io_defang_v1_fabric_pb = require('../../../io/defang/v1/fabric_pb.js');
var google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js');
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');

function serialize_google_protobuf_Empty(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_google_protobuf_Empty(buffer_arg) {
  return google_protobuf_empty_pb.Empty.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_CanIUseRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_CanIUseRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.CanIUseRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_CanIUseResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_CanIUseResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.CanIUseResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DebugRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DebugRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DebugRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DebugResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DebugResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.DebugResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DelegateSubdomainZoneRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DelegateSubdomainZoneRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DelegateSubdomainZoneRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DelegateSubdomainZoneResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DelegateSubdomainZoneResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DeleteConfigsRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DeleteConfigsRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DeleteConfigsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DeleteRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DeleteRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DeleteRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DeleteResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DeleteResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.DeleteResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DeployRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DeployRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DeployRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DeployResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DeployResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.DeployResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DestroyRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DestroyRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.DestroyRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_DestroyResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_DestroyResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.DestroyResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GenerateFilesRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GenerateFilesRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.GenerateFilesRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GenerateFilesResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GenerateFilesResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.GenerateFilesResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GenerateStatusRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GenerateStatusRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.GenerateStatusRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GetConfigsRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GetConfigsRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.GetConfigsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GetConfigsResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GetConfigsResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.GetConfigsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GetRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GetRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.GetRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GetSelectedProviderRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GetSelectedProviderRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.GetSelectedProviderRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GetSelectedProviderResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GetSelectedProviderResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.GetSelectedProviderResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GetServicesRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GetServicesRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.GetServicesRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_GetServicesResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_GetServicesResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.GetServicesResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_ListConfigsRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_ListConfigsRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.ListConfigsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_ListConfigsResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_ListConfigsResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.ListConfigsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_ListDeploymentsRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_ListDeploymentsRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.ListDeploymentsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_ListDeploymentsResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_ListDeploymentsResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.ListDeploymentsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_PublishRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_PublishRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.PublishRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_PutConfigRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_PutConfigRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.PutConfigRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_PutDeploymentRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_PutDeploymentRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.PutDeploymentRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Secrets(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Secrets(buffer_arg) {
  return io_defang_v1_fabric_pb.Secrets.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Service(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Service(buffer_arg) {
  return io_defang_v1_fabric_pb.Service.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_ServiceInfo(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_ServiceInfo(buffer_arg) {
  return io_defang_v1_fabric_pb.ServiceInfo.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_SetOptionsRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_SetOptionsRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.SetOptionsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_SetSelectedProviderRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_SetSelectedProviderRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.SetSelectedProviderRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_StartGenerateResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_StartGenerateResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.StartGenerateResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Status(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Status(buffer_arg) {
  return io_defang_v1_fabric_pb.Status.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_SubscribeRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_SubscribeRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.SubscribeRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_SubscribeResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_SubscribeResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.SubscribeResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TailRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TailRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.TailRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TailResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TailResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.TailResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TokenRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TokenRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.TokenRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TokenResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TokenResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.TokenResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_TrackRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_TrackRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.TrackRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_UploadURLRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_UploadURLRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.UploadURLRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_UploadURLResponse(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_UploadURLResponse(buffer_arg) {
  return io_defang_v1_fabric_pb.UploadURLResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_VerifyDNSSetupRequest(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_VerifyDNSSetupRequest(buffer_arg) {
  return io_defang_v1_fabric_pb.VerifyDNSSetupRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_Version(arg) {
  return Buffer.from(arg.serializeBinary());
}

function deserialize_io_defang_v1_Version(buffer_arg) {
  return io_defang_v1_fabric_pb.Version.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_io_defang_v1_WhoAmIResponse(arg) {
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
    requestType: io_defang_v1_fabric_pb.GetRequest,
    responseType: io_defang_v1_fabric_pb.ServiceInfo,
    requestSerialize: serialize_io_defang_v1_GetRequest,
    requestDeserialize: deserialize_io_defang_v1_GetRequest,
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
  destroy: {
    path: '/io.defang.v1.FabricController/Destroy',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.DestroyRequest,
    responseType: io_defang_v1_fabric_pb.DestroyResponse,
    requestSerialize: serialize_io_defang_v1_DestroyRequest,
    requestDeserialize: deserialize_io_defang_v1_DestroyRequest,
    responseSerialize: serialize_io_defang_v1_DestroyResponse,
    responseDeserialize: deserialize_io_defang_v1_DestroyResponse,
  },
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
    requestType: io_defang_v1_fabric_pb.GetServicesRequest,
    responseType: io_defang_v1_fabric_pb.GetServicesResponse,
    requestSerialize: serialize_io_defang_v1_GetServicesRequest,
    requestDeserialize: deserialize_io_defang_v1_GetServicesRequest,
    responseSerialize: serialize_io_defang_v1_GetServicesResponse,
    responseDeserialize: deserialize_io_defang_v1_GetServicesResponse,
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
  // deprecate - change to use *Config functions
putSecret: {
    path: '/io.defang.v1.FabricController/PutSecret',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.PutConfigRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_PutConfigRequest,
    requestDeserialize: deserialize_io_defang_v1_PutConfigRequest,
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
    requestType: io_defang_v1_fabric_pb.ListConfigsRequest,
    responseType: io_defang_v1_fabric_pb.Secrets,
    requestSerialize: serialize_io_defang_v1_ListConfigsRequest,
    requestDeserialize: deserialize_io_defang_v1_ListConfigsRequest,
    responseSerialize: serialize_io_defang_v1_Secrets,
    responseDeserialize: deserialize_io_defang_v1_Secrets,
  },
  getConfigs: {
    path: '/io.defang.v1.FabricController/GetConfigs',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.GetConfigsRequest,
    responseType: io_defang_v1_fabric_pb.GetConfigsResponse,
    requestSerialize: serialize_io_defang_v1_GetConfigsRequest,
    requestDeserialize: deserialize_io_defang_v1_GetConfigsRequest,
    responseSerialize: serialize_io_defang_v1_GetConfigsResponse,
    responseDeserialize: deserialize_io_defang_v1_GetConfigsResponse,
  },
  putConfig: {
    path: '/io.defang.v1.FabricController/PutConfig',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.PutConfigRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_PutConfigRequest,
    requestDeserialize: deserialize_io_defang_v1_PutConfigRequest,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  deleteConfigs: {
    path: '/io.defang.v1.FabricController/DeleteConfigs',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.DeleteConfigsRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_DeleteConfigsRequest,
    requestDeserialize: deserialize_io_defang_v1_DeleteConfigsRequest,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  listConfigs: {
    path: '/io.defang.v1.FabricController/ListConfigs',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.ListConfigsRequest,
    responseType: io_defang_v1_fabric_pb.ListConfigsResponse,
    requestSerialize: serialize_io_defang_v1_ListConfigsRequest,
    requestDeserialize: deserialize_io_defang_v1_ListConfigsRequest,
    responseSerialize: serialize_io_defang_v1_ListConfigsResponse,
    responseDeserialize: deserialize_io_defang_v1_ListConfigsResponse,
  },
  putDeployment: {
    path: '/io.defang.v1.FabricController/PutDeployment',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.PutDeploymentRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_PutDeploymentRequest,
    requestDeserialize: deserialize_io_defang_v1_PutDeploymentRequest,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  listDeployments: {
    path: '/io.defang.v1.FabricController/ListDeployments',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.ListDeploymentsRequest,
    responseType: io_defang_v1_fabric_pb.ListDeploymentsResponse,
    requestSerialize: serialize_io_defang_v1_ListDeploymentsRequest,
    requestDeserialize: deserialize_io_defang_v1_ListDeploymentsRequest,
    responseSerialize: serialize_io_defang_v1_ListDeploymentsResponse,
    responseDeserialize: deserialize_io_defang_v1_ListDeploymentsResponse,
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
  setOptions: {
    path: '/io.defang.v1.FabricController/SetOptions',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.SetOptionsRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_SetOptionsRequest,
    requestDeserialize: deserialize_io_defang_v1_SetOptionsRequest,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
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
  // Endpoint for GDPR compliance
deleteMe: {
    path: '/io.defang.v1.FabricController/DeleteMe',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  verifyDNSSetup: {
    path: '/io.defang.v1.FabricController/VerifyDNSSetup',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.VerifyDNSSetupRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_VerifyDNSSetupRequest,
    requestDeserialize: deserialize_io_defang_v1_VerifyDNSSetupRequest,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  getSelectedProvider: {
    path: '/io.defang.v1.FabricController/GetSelectedProvider',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.GetSelectedProviderRequest,
    responseType: io_defang_v1_fabric_pb.GetSelectedProviderResponse,
    requestSerialize: serialize_io_defang_v1_GetSelectedProviderRequest,
    requestDeserialize: deserialize_io_defang_v1_GetSelectedProviderRequest,
    responseSerialize: serialize_io_defang_v1_GetSelectedProviderResponse,
    responseDeserialize: deserialize_io_defang_v1_GetSelectedProviderResponse,
  },
  setSelectedProvider: {
    path: '/io.defang.v1.FabricController/SetSelectedProvider',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.SetSelectedProviderRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_io_defang_v1_SetSelectedProviderRequest,
    requestDeserialize: deserialize_io_defang_v1_SetSelectedProviderRequest,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  canIUse: {
    path: '/io.defang.v1.FabricController/CanIUse',
    requestStream: false,
    responseStream: false,
    requestType: io_defang_v1_fabric_pb.CanIUseRequest,
    responseType: io_defang_v1_fabric_pb.CanIUseResponse,
    requestSerialize: serialize_io_defang_v1_CanIUseRequest,
    requestDeserialize: deserialize_io_defang_v1_CanIUseRequest,
    responseSerialize: serialize_io_defang_v1_CanIUseResponse,
    responseDeserialize: deserialize_io_defang_v1_CanIUseResponse,
  },
};

exports.FabricControllerClient = grpc.makeGenericClientConstructor(FabricControllerService, 'FabricController');
