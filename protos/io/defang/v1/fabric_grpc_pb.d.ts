// GENERATED CODE -- DO NOT EDIT!

// package: io.defang.v1
// file: io/defang/v1/fabric.proto

import * as io_defang_v1_fabric_pb from "../../../io/defang/v1/fabric_pb";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";
import * as grpc from "@grpc/grpc-js";

interface IFabricControllerService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
  getStatus: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.Status>;
  getVersion: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.Version>;
  token: grpc.MethodDefinition<io_defang_v1_fabric_pb.TokenRequest, io_defang_v1_fabric_pb.TokenResponse>;
  revokeToken: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, google_protobuf_empty_pb.Empty>;
  tail: grpc.MethodDefinition<io_defang_v1_fabric_pb.TailRequest, io_defang_v1_fabric_pb.TailResponse>;
  update: grpc.MethodDefinition<io_defang_v1_fabric_pb.Service, io_defang_v1_fabric_pb.ServiceInfo>;
  deploy: grpc.MethodDefinition<io_defang_v1_fabric_pb.DeployRequest, io_defang_v1_fabric_pb.DeployResponse>;
  get: grpc.MethodDefinition<io_defang_v1_fabric_pb.ServiceID, io_defang_v1_fabric_pb.ServiceInfo>;
  delete: grpc.MethodDefinition<io_defang_v1_fabric_pb.DeleteRequest, io_defang_v1_fabric_pb.DeleteResponse>;
  publish: grpc.MethodDefinition<io_defang_v1_fabric_pb.PublishRequest, google_protobuf_empty_pb.Empty>;
  subscribe: grpc.MethodDefinition<io_defang_v1_fabric_pb.SubscribeRequest, io_defang_v1_fabric_pb.SubscribeResponse>;
  getServices: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.ListServicesResponse>;
  generateFiles: grpc.MethodDefinition<io_defang_v1_fabric_pb.GenerateFilesRequest, io_defang_v1_fabric_pb.GenerateFilesResponse>;
  startGenerate: grpc.MethodDefinition<io_defang_v1_fabric_pb.GenerateFilesRequest, io_defang_v1_fabric_pb.StartGenerateResponse>;
  generateStatus: grpc.MethodDefinition<io_defang_v1_fabric_pb.GenerateStatusRequest, io_defang_v1_fabric_pb.GenerateFilesResponse>;
  debug: grpc.MethodDefinition<io_defang_v1_fabric_pb.DebugRequest, io_defang_v1_fabric_pb.DebugResponse>;
  signEULA: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, google_protobuf_empty_pb.Empty>;
  checkToS: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, google_protobuf_empty_pb.Empty>;
  putSecret: grpc.MethodDefinition<io_defang_v1_fabric_pb.SecretValue, google_protobuf_empty_pb.Empty>;
  deleteSecrets: grpc.MethodDefinition<io_defang_v1_fabric_pb.Secrets, google_protobuf_empty_pb.Empty>;
  listSecrets: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.Secrets>;
  createUploadURL: grpc.MethodDefinition<io_defang_v1_fabric_pb.UploadURLRequest, io_defang_v1_fabric_pb.UploadURLResponse>;
  delegateSubdomainZone: grpc.MethodDefinition<io_defang_v1_fabric_pb.DelegateSubdomainZoneRequest, io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>;
  deleteSubdomainZone: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, google_protobuf_empty_pb.Empty>;
  getDelegateSubdomainZone: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>;
  whoAmI: grpc.MethodDefinition<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.WhoAmIResponse>;
  track: grpc.MethodDefinition<io_defang_v1_fabric_pb.TrackRequest, google_protobuf_empty_pb.Empty>;
}

export const FabricControllerService: IFabricControllerService;

export interface IFabricControllerServer extends grpc.UntypedServiceImplementation {
  getStatus: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.Status>;
  getVersion: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.Version>;
  token: grpc.handleUnaryCall<io_defang_v1_fabric_pb.TokenRequest, io_defang_v1_fabric_pb.TokenResponse>;
  revokeToken: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, google_protobuf_empty_pb.Empty>;
  tail: grpc.handleServerStreamingCall<io_defang_v1_fabric_pb.TailRequest, io_defang_v1_fabric_pb.TailResponse>;
  update: grpc.handleUnaryCall<io_defang_v1_fabric_pb.Service, io_defang_v1_fabric_pb.ServiceInfo>;
  deploy: grpc.handleUnaryCall<io_defang_v1_fabric_pb.DeployRequest, io_defang_v1_fabric_pb.DeployResponse>;
  get: grpc.handleUnaryCall<io_defang_v1_fabric_pb.ServiceID, io_defang_v1_fabric_pb.ServiceInfo>;
  delete: grpc.handleUnaryCall<io_defang_v1_fabric_pb.DeleteRequest, io_defang_v1_fabric_pb.DeleteResponse>;
  publish: grpc.handleUnaryCall<io_defang_v1_fabric_pb.PublishRequest, google_protobuf_empty_pb.Empty>;
  subscribe: grpc.handleServerStreamingCall<io_defang_v1_fabric_pb.SubscribeRequest, io_defang_v1_fabric_pb.SubscribeResponse>;
  getServices: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.ListServicesResponse>;
  generateFiles: grpc.handleUnaryCall<io_defang_v1_fabric_pb.GenerateFilesRequest, io_defang_v1_fabric_pb.GenerateFilesResponse>;
  startGenerate: grpc.handleUnaryCall<io_defang_v1_fabric_pb.GenerateFilesRequest, io_defang_v1_fabric_pb.StartGenerateResponse>;
  generateStatus: grpc.handleUnaryCall<io_defang_v1_fabric_pb.GenerateStatusRequest, io_defang_v1_fabric_pb.GenerateFilesResponse>;
  debug: grpc.handleUnaryCall<io_defang_v1_fabric_pb.DebugRequest, io_defang_v1_fabric_pb.DebugResponse>;
  signEULA: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, google_protobuf_empty_pb.Empty>;
  checkToS: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, google_protobuf_empty_pb.Empty>;
  putSecret: grpc.handleUnaryCall<io_defang_v1_fabric_pb.SecretValue, google_protobuf_empty_pb.Empty>;
  deleteSecrets: grpc.handleUnaryCall<io_defang_v1_fabric_pb.Secrets, google_protobuf_empty_pb.Empty>;
  listSecrets: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.Secrets>;
  createUploadURL: grpc.handleUnaryCall<io_defang_v1_fabric_pb.UploadURLRequest, io_defang_v1_fabric_pb.UploadURLResponse>;
  delegateSubdomainZone: grpc.handleUnaryCall<io_defang_v1_fabric_pb.DelegateSubdomainZoneRequest, io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>;
  deleteSubdomainZone: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, google_protobuf_empty_pb.Empty>;
  getDelegateSubdomainZone: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>;
  whoAmI: grpc.handleUnaryCall<google_protobuf_empty_pb.Empty, io_defang_v1_fabric_pb.WhoAmIResponse>;
  track: grpc.handleUnaryCall<io_defang_v1_fabric_pb.TrackRequest, google_protobuf_empty_pb.Empty>;
}

export class FabricControllerClient extends grpc.Client {
  constructor(address: string, credentials: grpc.ChannelCredentials, options?: object);
  getStatus(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<io_defang_v1_fabric_pb.Status>): grpc.ClientUnaryCall;
  getStatus(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.Status>): grpc.ClientUnaryCall;
  getStatus(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.Status>): grpc.ClientUnaryCall;
  getVersion(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<io_defang_v1_fabric_pb.Version>): grpc.ClientUnaryCall;
  getVersion(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.Version>): grpc.ClientUnaryCall;
  getVersion(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.Version>): grpc.ClientUnaryCall;
  token(argument: io_defang_v1_fabric_pb.TokenRequest, callback: grpc.requestCallback<io_defang_v1_fabric_pb.TokenResponse>): grpc.ClientUnaryCall;
  token(argument: io_defang_v1_fabric_pb.TokenRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.TokenResponse>): grpc.ClientUnaryCall;
  token(argument: io_defang_v1_fabric_pb.TokenRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.TokenResponse>): grpc.ClientUnaryCall;
  revokeToken(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  revokeToken(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  revokeToken(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  tail(argument: io_defang_v1_fabric_pb.TailRequest, metadataOrOptions?: grpc.Metadata | grpc.CallOptions | null): grpc.ClientReadableStream<io_defang_v1_fabric_pb.TailResponse>;
  tail(argument: io_defang_v1_fabric_pb.TailRequest, metadata?: grpc.Metadata | null, options?: grpc.CallOptions | null): grpc.ClientReadableStream<io_defang_v1_fabric_pb.TailResponse>;
  update(argument: io_defang_v1_fabric_pb.Service, callback: grpc.requestCallback<io_defang_v1_fabric_pb.ServiceInfo>): grpc.ClientUnaryCall;
  update(argument: io_defang_v1_fabric_pb.Service, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.ServiceInfo>): grpc.ClientUnaryCall;
  update(argument: io_defang_v1_fabric_pb.Service, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.ServiceInfo>): grpc.ClientUnaryCall;
  deploy(argument: io_defang_v1_fabric_pb.DeployRequest, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DeployResponse>): grpc.ClientUnaryCall;
  deploy(argument: io_defang_v1_fabric_pb.DeployRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DeployResponse>): grpc.ClientUnaryCall;
  deploy(argument: io_defang_v1_fabric_pb.DeployRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DeployResponse>): grpc.ClientUnaryCall;
  get(argument: io_defang_v1_fabric_pb.ServiceID, callback: grpc.requestCallback<io_defang_v1_fabric_pb.ServiceInfo>): grpc.ClientUnaryCall;
  get(argument: io_defang_v1_fabric_pb.ServiceID, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.ServiceInfo>): grpc.ClientUnaryCall;
  get(argument: io_defang_v1_fabric_pb.ServiceID, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.ServiceInfo>): grpc.ClientUnaryCall;
  delete(argument: io_defang_v1_fabric_pb.DeleteRequest, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DeleteResponse>): grpc.ClientUnaryCall;
  delete(argument: io_defang_v1_fabric_pb.DeleteRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DeleteResponse>): grpc.ClientUnaryCall;
  delete(argument: io_defang_v1_fabric_pb.DeleteRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DeleteResponse>): grpc.ClientUnaryCall;
  publish(argument: io_defang_v1_fabric_pb.PublishRequest, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  publish(argument: io_defang_v1_fabric_pb.PublishRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  publish(argument: io_defang_v1_fabric_pb.PublishRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  subscribe(argument: io_defang_v1_fabric_pb.SubscribeRequest, metadataOrOptions?: grpc.Metadata | grpc.CallOptions | null): grpc.ClientReadableStream<io_defang_v1_fabric_pb.SubscribeResponse>;
  subscribe(argument: io_defang_v1_fabric_pb.SubscribeRequest, metadata?: grpc.Metadata | null, options?: grpc.CallOptions | null): grpc.ClientReadableStream<io_defang_v1_fabric_pb.SubscribeResponse>;
  getServices(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<io_defang_v1_fabric_pb.ListServicesResponse>): grpc.ClientUnaryCall;
  getServices(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.ListServicesResponse>): grpc.ClientUnaryCall;
  getServices(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.ListServicesResponse>): grpc.ClientUnaryCall;
  generateFiles(argument: io_defang_v1_fabric_pb.GenerateFilesRequest, callback: grpc.requestCallback<io_defang_v1_fabric_pb.GenerateFilesResponse>): grpc.ClientUnaryCall;
  generateFiles(argument: io_defang_v1_fabric_pb.GenerateFilesRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.GenerateFilesResponse>): grpc.ClientUnaryCall;
  generateFiles(argument: io_defang_v1_fabric_pb.GenerateFilesRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.GenerateFilesResponse>): grpc.ClientUnaryCall;
  startGenerate(argument: io_defang_v1_fabric_pb.GenerateFilesRequest, callback: grpc.requestCallback<io_defang_v1_fabric_pb.StartGenerateResponse>): grpc.ClientUnaryCall;
  startGenerate(argument: io_defang_v1_fabric_pb.GenerateFilesRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.StartGenerateResponse>): grpc.ClientUnaryCall;
  startGenerate(argument: io_defang_v1_fabric_pb.GenerateFilesRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.StartGenerateResponse>): grpc.ClientUnaryCall;
  generateStatus(argument: io_defang_v1_fabric_pb.GenerateStatusRequest, callback: grpc.requestCallback<io_defang_v1_fabric_pb.GenerateFilesResponse>): grpc.ClientUnaryCall;
  generateStatus(argument: io_defang_v1_fabric_pb.GenerateStatusRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.GenerateFilesResponse>): grpc.ClientUnaryCall;
  generateStatus(argument: io_defang_v1_fabric_pb.GenerateStatusRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.GenerateFilesResponse>): grpc.ClientUnaryCall;
  debug(argument: io_defang_v1_fabric_pb.DebugRequest, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DebugResponse>): grpc.ClientUnaryCall;
  debug(argument: io_defang_v1_fabric_pb.DebugRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DebugResponse>): grpc.ClientUnaryCall;
  debug(argument: io_defang_v1_fabric_pb.DebugRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DebugResponse>): grpc.ClientUnaryCall;
  signEULA(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  signEULA(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  signEULA(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  checkToS(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  checkToS(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  checkToS(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  putSecret(argument: io_defang_v1_fabric_pb.SecretValue, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  putSecret(argument: io_defang_v1_fabric_pb.SecretValue, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  putSecret(argument: io_defang_v1_fabric_pb.SecretValue, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  deleteSecrets(argument: io_defang_v1_fabric_pb.Secrets, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  deleteSecrets(argument: io_defang_v1_fabric_pb.Secrets, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  deleteSecrets(argument: io_defang_v1_fabric_pb.Secrets, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  listSecrets(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<io_defang_v1_fabric_pb.Secrets>): grpc.ClientUnaryCall;
  listSecrets(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.Secrets>): grpc.ClientUnaryCall;
  listSecrets(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.Secrets>): grpc.ClientUnaryCall;
  createUploadURL(argument: io_defang_v1_fabric_pb.UploadURLRequest, callback: grpc.requestCallback<io_defang_v1_fabric_pb.UploadURLResponse>): grpc.ClientUnaryCall;
  createUploadURL(argument: io_defang_v1_fabric_pb.UploadURLRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.UploadURLResponse>): grpc.ClientUnaryCall;
  createUploadURL(argument: io_defang_v1_fabric_pb.UploadURLRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.UploadURLResponse>): grpc.ClientUnaryCall;
  delegateSubdomainZone(argument: io_defang_v1_fabric_pb.DelegateSubdomainZoneRequest, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>): grpc.ClientUnaryCall;
  delegateSubdomainZone(argument: io_defang_v1_fabric_pb.DelegateSubdomainZoneRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>): grpc.ClientUnaryCall;
  delegateSubdomainZone(argument: io_defang_v1_fabric_pb.DelegateSubdomainZoneRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>): grpc.ClientUnaryCall;
  deleteSubdomainZone(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  deleteSubdomainZone(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  deleteSubdomainZone(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  getDelegateSubdomainZone(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>): grpc.ClientUnaryCall;
  getDelegateSubdomainZone(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>): grpc.ClientUnaryCall;
  getDelegateSubdomainZone(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.DelegateSubdomainZoneResponse>): grpc.ClientUnaryCall;
  whoAmI(argument: google_protobuf_empty_pb.Empty, callback: grpc.requestCallback<io_defang_v1_fabric_pb.WhoAmIResponse>): grpc.ClientUnaryCall;
  whoAmI(argument: google_protobuf_empty_pb.Empty, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.WhoAmIResponse>): grpc.ClientUnaryCall;
  whoAmI(argument: google_protobuf_empty_pb.Empty, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<io_defang_v1_fabric_pb.WhoAmIResponse>): grpc.ClientUnaryCall;
  track(argument: io_defang_v1_fabric_pb.TrackRequest, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  track(argument: io_defang_v1_fabric_pb.TrackRequest, metadataOrOptions: grpc.Metadata | grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
  track(argument: io_defang_v1_fabric_pb.TrackRequest, metadata: grpc.Metadata | null, options: grpc.CallOptions | null, callback: grpc.requestCallback<google_protobuf_empty_pb.Empty>): grpc.ClientUnaryCall;
}
