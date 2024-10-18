// package: io.defang.v1
// file: io/defang/v1/fabric.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

export class VerifyDNSSetupRequest extends jspb.Message {
  getDomain(): string;
  setDomain(value: string): void;

  clearTargetsList(): void;
  getTargetsList(): Array<string>;
  setTargetsList(value: Array<string>): void;
  addTargets(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): VerifyDNSSetupRequest.AsObject;
  static toObject(includeInstance: boolean, msg: VerifyDNSSetupRequest): VerifyDNSSetupRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: VerifyDNSSetupRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): VerifyDNSSetupRequest;
  static deserializeBinaryFromReader(message: VerifyDNSSetupRequest, reader: jspb.BinaryReader): VerifyDNSSetupRequest;
}

export namespace VerifyDNSSetupRequest {
  export type AsObject = {
    domain: string,
    targetsList: Array<string>,
  }
}

export class DestroyRequest extends jspb.Message {
  getProject(): string;
  setProject(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DestroyRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DestroyRequest): DestroyRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DestroyRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DestroyRequest;
  static deserializeBinaryFromReader(message: DestroyRequest, reader: jspb.BinaryReader): DestroyRequest;
}

export namespace DestroyRequest {
  export type AsObject = {
    project: string,
  }
}

export class DestroyResponse extends jspb.Message {
  getEtag(): string;
  setEtag(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DestroyResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DestroyResponse): DestroyResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DestroyResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DestroyResponse;
  static deserializeBinaryFromReader(message: DestroyResponse, reader: jspb.BinaryReader): DestroyResponse;
}

export namespace DestroyResponse {
  export type AsObject = {
    etag: string,
  }
}

export class DebugRequest extends jspb.Message {
  clearFilesList(): void;
  getFilesList(): Array<File>;
  setFilesList(value: Array<File>): void;
  addFiles(value?: File, index?: number): File;

  getEtag(): string;
  setEtag(value: string): void;

  getProject(): string;
  setProject(value: string): void;

  getLogs(): string;
  setLogs(value: string): void;

  clearServicesList(): void;
  getServicesList(): Array<string>;
  setServicesList(value: Array<string>): void;
  addServices(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DebugRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DebugRequest): DebugRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DebugRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DebugRequest;
  static deserializeBinaryFromReader(message: DebugRequest, reader: jspb.BinaryReader): DebugRequest;
}

export namespace DebugRequest {
  export type AsObject = {
    filesList: Array<File.AsObject>,
    etag: string,
    project: string,
    logs: string,
    servicesList: Array<string>,
  }
}

export class DebugResponse extends jspb.Message {
  getGeneral(): string;
  setGeneral(value: string): void;

  clearIssuesList(): void;
  getIssuesList(): Array<Issue>;
  setIssuesList(value: Array<Issue>): void;
  addIssues(value?: Issue, index?: number): Issue;

  clearRequestsList(): void;
  getRequestsList(): Array<string>;
  setRequestsList(value: Array<string>): void;
  addRequests(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DebugResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DebugResponse): DebugResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DebugResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DebugResponse;
  static deserializeBinaryFromReader(message: DebugResponse, reader: jspb.BinaryReader): DebugResponse;
}

export namespace DebugResponse {
  export type AsObject = {
    general: string,
    issuesList: Array<Issue.AsObject>,
    requestsList: Array<string>,
  }
}

export class Issue extends jspb.Message {
  getType(): string;
  setType(value: string): void;

  getSeverity(): string;
  setSeverity(value: string): void;

  getDetails(): string;
  setDetails(value: string): void;

  clearCodeChangesList(): void;
  getCodeChangesList(): Array<CodeChange>;
  setCodeChangesList(value: Array<CodeChange>): void;
  addCodeChanges(value?: CodeChange, index?: number): CodeChange;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Issue.AsObject;
  static toObject(includeInstance: boolean, msg: Issue): Issue.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Issue, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Issue;
  static deserializeBinaryFromReader(message: Issue, reader: jspb.BinaryReader): Issue;
}

export namespace Issue {
  export type AsObject = {
    type: string,
    severity: string,
    details: string,
    codeChangesList: Array<CodeChange.AsObject>,
  }
}

export class CodeChange extends jspb.Message {
  getFile(): string;
  setFile(value: string): void;

  getChange(): string;
  setChange(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CodeChange.AsObject;
  static toObject(includeInstance: boolean, msg: CodeChange): CodeChange.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CodeChange, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CodeChange;
  static deserializeBinaryFromReader(message: CodeChange, reader: jspb.BinaryReader): CodeChange;
}

export namespace CodeChange {
  export type AsObject = {
    file: string,
    change: string,
  }
}

export class TrackRequest extends jspb.Message {
  getAnonId(): string;
  setAnonId(value: string): void;

  getEvent(): string;
  setEvent(value: string): void;

  getPropertiesMap(): jspb.Map<string, string>;
  clearPropertiesMap(): void;
  getOs(): string;
  setOs(value: string): void;

  getArch(): string;
  setArch(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TrackRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TrackRequest): TrackRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TrackRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TrackRequest;
  static deserializeBinaryFromReader(message: TrackRequest, reader: jspb.BinaryReader): TrackRequest;
}

export namespace TrackRequest {
  export type AsObject = {
    anonId: string,
    event: string,
    propertiesMap: Array<[string, string]>,
    os: string,
    arch: string,
  }
}

export class DeployRequest extends jspb.Message {
  clearServicesList(): void;
  getServicesList(): Array<Service>;
  setServicesList(value: Array<Service>): void;
  addServices(value?: Service, index?: number): Service;

  getProject(): string;
  setProject(value: string): void;

  getMode(): DeploymentModeMap[keyof DeploymentModeMap];
  setMode(value: DeploymentModeMap[keyof DeploymentModeMap]): void;

  getCompose(): Uint8Array | string;
  getCompose_asU8(): Uint8Array;
  getCompose_asB64(): string;
  setCompose(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeployRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeployRequest): DeployRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeployRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeployRequest;
  static deserializeBinaryFromReader(message: DeployRequest, reader: jspb.BinaryReader): DeployRequest;
}

export namespace DeployRequest {
  export type AsObject = {
    servicesList: Array<Service.AsObject>,
    project: string,
    mode: DeploymentModeMap[keyof DeploymentModeMap],
    compose: Uint8Array | string,
  }
}

export class DeployResponse extends jspb.Message {
  clearServicesList(): void;
  getServicesList(): Array<ServiceInfo>;
  setServicesList(value: Array<ServiceInfo>): void;
  addServices(value?: ServiceInfo, index?: number): ServiceInfo;

  getEtag(): string;
  setEtag(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeployResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeployResponse): DeployResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeployResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeployResponse;
  static deserializeBinaryFromReader(message: DeployResponse, reader: jspb.BinaryReader): DeployResponse;
}

export namespace DeployResponse {
  export type AsObject = {
    servicesList: Array<ServiceInfo.AsObject>,
    etag: string,
  }
}

export class DeleteRequest extends jspb.Message {
  clearNamesList(): void;
  getNamesList(): Array<string>;
  setNamesList(value: Array<string>): void;
  addNames(value: string, index?: number): string;

  getProject(): string;
  setProject(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteRequest): DeleteRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteRequest;
  static deserializeBinaryFromReader(message: DeleteRequest, reader: jspb.BinaryReader): DeleteRequest;
}

export namespace DeleteRequest {
  export type AsObject = {
    namesList: Array<string>,
    project: string,
  }
}

export class DeleteResponse extends jspb.Message {
  getEtag(): string;
  setEtag(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteResponse): DeleteResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteResponse;
  static deserializeBinaryFromReader(message: DeleteResponse, reader: jspb.BinaryReader): DeleteResponse;
}

export namespace DeleteResponse {
  export type AsObject = {
    etag: string,
  }
}

export class GenerateFilesRequest extends jspb.Message {
  getPrompt(): string;
  setPrompt(value: string): void;

  getLanguage(): string;
  setLanguage(value: string): void;

  getAgreeTos(): boolean;
  setAgreeTos(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateFilesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateFilesRequest): GenerateFilesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GenerateFilesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateFilesRequest;
  static deserializeBinaryFromReader(message: GenerateFilesRequest, reader: jspb.BinaryReader): GenerateFilesRequest;
}

export namespace GenerateFilesRequest {
  export type AsObject = {
    prompt: string,
    language: string,
    agreeTos: boolean,
  }
}

export class File extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getContent(): string;
  setContent(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): File.AsObject;
  static toObject(includeInstance: boolean, msg: File): File.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: File, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): File;
  static deserializeBinaryFromReader(message: File, reader: jspb.BinaryReader): File;
}

export namespace File {
  export type AsObject = {
    name: string,
    content: string,
  }
}

export class GenerateFilesResponse extends jspb.Message {
  clearFilesList(): void;
  getFilesList(): Array<File>;
  setFilesList(value: Array<File>): void;
  addFiles(value?: File, index?: number): File;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateFilesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateFilesResponse): GenerateFilesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GenerateFilesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateFilesResponse;
  static deserializeBinaryFromReader(message: GenerateFilesResponse, reader: jspb.BinaryReader): GenerateFilesResponse;
}

export namespace GenerateFilesResponse {
  export type AsObject = {
    filesList: Array<File.AsObject>,
  }
}

export class StartGenerateResponse extends jspb.Message {
  getUuid(): string;
  setUuid(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartGenerateResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StartGenerateResponse): StartGenerateResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StartGenerateResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartGenerateResponse;
  static deserializeBinaryFromReader(message: StartGenerateResponse, reader: jspb.BinaryReader): StartGenerateResponse;
}

export namespace StartGenerateResponse {
  export type AsObject = {
    uuid: string,
  }
}

export class GenerateStatusRequest extends jspb.Message {
  getUuid(): string;
  setUuid(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateStatusRequest): GenerateStatusRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GenerateStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateStatusRequest;
  static deserializeBinaryFromReader(message: GenerateStatusRequest, reader: jspb.BinaryReader): GenerateStatusRequest;
}

export namespace GenerateStatusRequest {
  export type AsObject = {
    uuid: string,
  }
}

export class UploadURLRequest extends jspb.Message {
  getDigest(): string;
  setDigest(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UploadURLRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UploadURLRequest): UploadURLRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UploadURLRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UploadURLRequest;
  static deserializeBinaryFromReader(message: UploadURLRequest, reader: jspb.BinaryReader): UploadURLRequest;
}

export namespace UploadURLRequest {
  export type AsObject = {
    digest: string,
  }
}

export class UploadURLResponse extends jspb.Message {
  getUrl(): string;
  setUrl(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UploadURLResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UploadURLResponse): UploadURLResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UploadURLResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UploadURLResponse;
  static deserializeBinaryFromReader(message: UploadURLResponse, reader: jspb.BinaryReader): UploadURLResponse;
}

export namespace UploadURLResponse {
  export type AsObject = {
    url: string,
  }
}

export class ServiceInfo extends jspb.Message {
  hasService(): boolean;
  clearService(): void;
  getService(): Service | undefined;
  setService(value?: Service): void;

  clearEndpointsList(): void;
  getEndpointsList(): Array<string>;
  setEndpointsList(value: Array<string>): void;
  addEndpoints(value: string, index?: number): string;

  getProject(): string;
  setProject(value: string): void;

  getEtag(): string;
  setEtag(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  clearNatIpsList(): void;
  getNatIpsList(): Array<string>;
  setNatIpsList(value: Array<string>): void;
  addNatIps(value: string, index?: number): string;

  clearLbIpsList(): void;
  getLbIpsList(): Array<string>;
  setLbIpsList(value: Array<string>): void;
  addLbIps(value: string, index?: number): string;

  getPrivateFqdn(): string;
  setPrivateFqdn(value: string): void;

  getPublicFqdn(): string;
  setPublicFqdn(value: string): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getZoneId(): string;
  setZoneId(value: string): void;

  getUseAcmeCert(): boolean;
  setUseAcmeCert(value: boolean): void;

  getState(): ServiceStateMap[keyof ServiceStateMap];
  setState(value: ServiceStateMap[keyof ServiceStateMap]): void;

  getDomainname(): string;
  setDomainname(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ServiceInfo.AsObject;
  static toObject(includeInstance: boolean, msg: ServiceInfo): ServiceInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ServiceInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ServiceInfo;
  static deserializeBinaryFromReader(message: ServiceInfo, reader: jspb.BinaryReader): ServiceInfo;
}

export namespace ServiceInfo {
  export type AsObject = {
    service?: Service.AsObject,
    endpointsList: Array<string>,
    project: string,
    etag: string,
    status: string,
    natIpsList: Array<string>,
    lbIpsList: Array<string>,
    privateFqdn: string,
    publicFqdn: string,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    zoneId: string,
    useAcmeCert: boolean,
    state: ServiceStateMap[keyof ServiceStateMap],
    domainname: string,
  }
}

export class Secrets extends jspb.Message {
  clearNamesList(): void;
  getNamesList(): Array<string>;
  setNamesList(value: Array<string>): void;
  addNames(value: string, index?: number): string;

  getProject(): string;
  setProject(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Secrets.AsObject;
  static toObject(includeInstance: boolean, msg: Secrets): Secrets.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Secrets, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Secrets;
  static deserializeBinaryFromReader(message: Secrets, reader: jspb.BinaryReader): Secrets;
}

export namespace Secrets {
  export type AsObject = {
    namesList: Array<string>,
    project: string,
  }
}

export class SecretValue extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getValue(): string;
  setValue(value: string): void;

  getProject(): string;
  setProject(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SecretValue.AsObject;
  static toObject(includeInstance: boolean, msg: SecretValue): SecretValue.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SecretValue, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SecretValue;
  static deserializeBinaryFromReader(message: SecretValue, reader: jspb.BinaryReader): SecretValue;
}

export namespace SecretValue {
  export type AsObject = {
    name: string,
    value: string,
    project: string,
  }
}

export class Config extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getValue(): string;
  setValue(value: string): void;

  getProject(): string;
  setProject(value: string): void;

  getType(): ConfigTypeMap[keyof ConfigTypeMap];
  setType(value: ConfigTypeMap[keyof ConfigTypeMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Config.AsObject;
  static toObject(includeInstance: boolean, msg: Config): Config.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Config, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Config;
  static deserializeBinaryFromReader(message: Config, reader: jspb.BinaryReader): Config;
}

export namespace Config {
  export type AsObject = {
    name: string,
    value: string,
    project: string,
    type: ConfigTypeMap[keyof ConfigTypeMap],
  }
}

export class ConfigKey extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getProject(): string;
  setProject(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ConfigKey.AsObject;
  static toObject(includeInstance: boolean, msg: ConfigKey): ConfigKey.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ConfigKey, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ConfigKey;
  static deserializeBinaryFromReader(message: ConfigKey, reader: jspb.BinaryReader): ConfigKey;
}

export namespace ConfigKey {
  export type AsObject = {
    name: string,
    project: string,
  }
}

export class PutConfigRequest extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getValue(): string;
  setValue(value: string): void;

  getProject(): string;
  setProject(value: string): void;

  getType(): ConfigTypeMap[keyof ConfigTypeMap];
  setType(value: ConfigTypeMap[keyof ConfigTypeMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutConfigRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutConfigRequest): PutConfigRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutConfigRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutConfigRequest;
  static deserializeBinaryFromReader(message: PutConfigRequest, reader: jspb.BinaryReader): PutConfigRequest;
}

export namespace PutConfigRequest {
  export type AsObject = {
    name: string,
    value: string,
    project: string,
    type: ConfigTypeMap[keyof ConfigTypeMap],
  }
}

export class GetConfigsRequest extends jspb.Message {
  clearConfigsList(): void;
  getConfigsList(): Array<ConfigKey>;
  setConfigsList(value: Array<ConfigKey>): void;
  addConfigs(value?: ConfigKey, index?: number): ConfigKey;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetConfigsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetConfigsRequest): GetConfigsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetConfigsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetConfigsRequest;
  static deserializeBinaryFromReader(message: GetConfigsRequest, reader: jspb.BinaryReader): GetConfigsRequest;
}

export namespace GetConfigsRequest {
  export type AsObject = {
    configsList: Array<ConfigKey.AsObject>,
  }
}

export class GetConfigsResponse extends jspb.Message {
  clearConfigsList(): void;
  getConfigsList(): Array<Config>;
  setConfigsList(value: Array<Config>): void;
  addConfigs(value?: Config, index?: number): Config;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetConfigsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetConfigsResponse): GetConfigsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetConfigsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetConfigsResponse;
  static deserializeBinaryFromReader(message: GetConfigsResponse, reader: jspb.BinaryReader): GetConfigsResponse;
}

export namespace GetConfigsResponse {
  export type AsObject = {
    configsList: Array<Config.AsObject>,
  }
}

export class DeleteConfigsRequest extends jspb.Message {
  clearConfigsList(): void;
  getConfigsList(): Array<ConfigKey>;
  setConfigsList(value: Array<ConfigKey>): void;
  addConfigs(value?: ConfigKey, index?: number): ConfigKey;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteConfigsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteConfigsRequest): DeleteConfigsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteConfigsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteConfigsRequest;
  static deserializeBinaryFromReader(message: DeleteConfigsRequest, reader: jspb.BinaryReader): DeleteConfigsRequest;
}

export namespace DeleteConfigsRequest {
  export type AsObject = {
    configsList: Array<ConfigKey.AsObject>,
  }
}

export class ListConfigsRequest extends jspb.Message {
  getProject(): string;
  setProject(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListConfigsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListConfigsRequest): ListConfigsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListConfigsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListConfigsRequest;
  static deserializeBinaryFromReader(message: ListConfigsRequest, reader: jspb.BinaryReader): ListConfigsRequest;
}

export namespace ListConfigsRequest {
  export type AsObject = {
    project: string,
  }
}

export class ListConfigsResponse extends jspb.Message {
  clearConfigsList(): void;
  getConfigsList(): Array<ConfigKey>;
  setConfigsList(value: Array<ConfigKey>): void;
  addConfigs(value?: ConfigKey, index?: number): ConfigKey;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListConfigsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListConfigsResponse): ListConfigsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListConfigsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListConfigsResponse;
  static deserializeBinaryFromReader(message: ListConfigsResponse, reader: jspb.BinaryReader): ListConfigsResponse;
}

export namespace ListConfigsResponse {
  export type AsObject = {
    configsList: Array<ConfigKey.AsObject>,
  }
}

export class TokenRequest extends jspb.Message {
  getTenant(): string;
  setTenant(value: string): void;

  getAuthCode(): string;
  setAuthCode(value: string): void;

  clearScopeList(): void;
  getScopeList(): Array<string>;
  setScopeList(value: Array<string>): void;
  addScope(value: string, index?: number): string;

  getAssertion(): string;
  setAssertion(value: string): void;

  getExpiresIn(): number;
  setExpiresIn(value: number): void;

  getAnonId(): string;
  setAnonId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TokenRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TokenRequest): TokenRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TokenRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TokenRequest;
  static deserializeBinaryFromReader(message: TokenRequest, reader: jspb.BinaryReader): TokenRequest;
}

export namespace TokenRequest {
  export type AsObject = {
    tenant: string,
    authCode: string,
    scopeList: Array<string>,
    assertion: string,
    expiresIn: number,
    anonId: string,
  }
}

export class TokenResponse extends jspb.Message {
  getAccessToken(): string;
  setAccessToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TokenResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TokenResponse): TokenResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TokenResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TokenResponse;
  static deserializeBinaryFromReader(message: TokenResponse, reader: jspb.BinaryReader): TokenResponse;
}

export namespace TokenResponse {
  export type AsObject = {
    accessToken: string,
  }
}

export class Status extends jspb.Message {
  getVersion(): string;
  setVersion(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Status.AsObject;
  static toObject(includeInstance: boolean, msg: Status): Status.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Status, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Status;
  static deserializeBinaryFromReader(message: Status, reader: jspb.BinaryReader): Status;
}

export namespace Status {
  export type AsObject = {
    version: string,
  }
}

export class Version extends jspb.Message {
  getFabric(): string;
  setFabric(value: string): void;

  getCliMin(): string;
  setCliMin(value: string): void;

  getPulumiMin(): string;
  setPulumiMin(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Version.AsObject;
  static toObject(includeInstance: boolean, msg: Version): Version.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Version, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Version;
  static deserializeBinaryFromReader(message: Version, reader: jspb.BinaryReader): Version;
}

export namespace Version {
  export type AsObject = {
    fabric: string,
    cliMin: string,
    pulumiMin: string,
  }
}

export class TailRequest extends jspb.Message {
  clearServicesList(): void;
  getServicesList(): Array<string>;
  setServicesList(value: Array<string>): void;
  addServices(value: string, index?: number): string;

  hasSince(): boolean;
  clearSince(): void;
  getSince(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setSince(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getEtag(): string;
  setEtag(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TailRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TailRequest): TailRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TailRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TailRequest;
  static deserializeBinaryFromReader(message: TailRequest, reader: jspb.BinaryReader): TailRequest;
}

export namespace TailRequest {
  export type AsObject = {
    servicesList: Array<string>,
    since?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    etag: string,
  }
}

export class LogEntry extends jspb.Message {
  getMessage(): string;
  setMessage(value: string): void;

  hasTimestamp(): boolean;
  clearTimestamp(): void;
  getTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getStderr(): boolean;
  setStderr(value: boolean): void;

  getService(): string;
  setService(value: string): void;

  getEtag(): string;
  setEtag(value: string): void;

  getHost(): string;
  setHost(value: string): void;

  getJob(): string;
  setJob(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LogEntry.AsObject;
  static toObject(includeInstance: boolean, msg: LogEntry): LogEntry.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: LogEntry, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LogEntry;
  static deserializeBinaryFromReader(message: LogEntry, reader: jspb.BinaryReader): LogEntry;
}

export namespace LogEntry {
  export type AsObject = {
    message: string,
    timestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    stderr: boolean,
    service: string,
    etag: string,
    host: string,
    job: string,
  }
}

export class TailResponse extends jspb.Message {
  clearEntriesList(): void;
  getEntriesList(): Array<LogEntry>;
  setEntriesList(value: Array<LogEntry>): void;
  addEntries(value?: LogEntry, index?: number): LogEntry;

  getService(): string;
  setService(value: string): void;

  getEtag(): string;
  setEtag(value: string): void;

  getHost(): string;
  setHost(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TailResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TailResponse): TailResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TailResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TailResponse;
  static deserializeBinaryFromReader(message: TailResponse, reader: jspb.BinaryReader): TailResponse;
}

export namespace TailResponse {
  export type AsObject = {
    entriesList: Array<LogEntry.AsObject>,
    service: string,
    etag: string,
    host: string,
  }
}

export class ListServicesResponse extends jspb.Message {
  clearServicesList(): void;
  getServicesList(): Array<ServiceInfo>;
  setServicesList(value: Array<ServiceInfo>): void;
  addServices(value?: ServiceInfo, index?: number): ServiceInfo;

  getProject(): string;
  setProject(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListServicesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListServicesResponse): ListServicesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListServicesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListServicesResponse;
  static deserializeBinaryFromReader(message: ListServicesResponse, reader: jspb.BinaryReader): ListServicesResponse;
}

export namespace ListServicesResponse {
  export type AsObject = {
    servicesList: Array<ServiceInfo.AsObject>,
    project: string,
  }
}

export class ProjectUpdate extends jspb.Message {
  clearServicesList(): void;
  getServicesList(): Array<ServiceInfo>;
  setServicesList(value: Array<ServiceInfo>): void;
  addServices(value?: ServiceInfo, index?: number): ServiceInfo;

  getAlbArn(): string;
  setAlbArn(value: string): void;

  getProject(): string;
  setProject(value: string): void;

  getCompose(): Uint8Array | string;
  getCompose_asU8(): Uint8Array;
  getCompose_asB64(): string;
  setCompose(value: Uint8Array | string): void;

  getCdVersion(): string;
  setCdVersion(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProjectUpdate.AsObject;
  static toObject(includeInstance: boolean, msg: ProjectUpdate): ProjectUpdate.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ProjectUpdate, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProjectUpdate;
  static deserializeBinaryFromReader(message: ProjectUpdate, reader: jspb.BinaryReader): ProjectUpdate;
}

export namespace ProjectUpdate {
  export type AsObject = {
    servicesList: Array<ServiceInfo.AsObject>,
    albArn: string,
    project: string,
    compose: Uint8Array | string,
    cdVersion: string,
  }
}

export class ServiceID extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ServiceID.AsObject;
  static toObject(includeInstance: boolean, msg: ServiceID): ServiceID.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ServiceID, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ServiceID;
  static deserializeBinaryFromReader(message: ServiceID, reader: jspb.BinaryReader): ServiceID;
}

export namespace ServiceID {
  export type AsObject = {
    name: string,
  }
}

export class Device extends jspb.Message {
  clearCapabilitiesList(): void;
  getCapabilitiesList(): Array<string>;
  setCapabilitiesList(value: Array<string>): void;
  addCapabilities(value: string, index?: number): string;

  getDriver(): string;
  setDriver(value: string): void;

  getCount(): number;
  setCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Device.AsObject;
  static toObject(includeInstance: boolean, msg: Device): Device.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Device, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Device;
  static deserializeBinaryFromReader(message: Device, reader: jspb.BinaryReader): Device;
}

export namespace Device {
  export type AsObject = {
    capabilitiesList: Array<string>,
    driver: string,
    count: number,
  }
}

export class Resource extends jspb.Message {
  getMemory(): number;
  setMemory(value: number): void;

  getCpus(): number;
  setCpus(value: number): void;

  clearDevicesList(): void;
  getDevicesList(): Array<Device>;
  setDevicesList(value: Array<Device>): void;
  addDevices(value?: Device, index?: number): Device;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Resource.AsObject;
  static toObject(includeInstance: boolean, msg: Resource): Resource.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Resource, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Resource;
  static deserializeBinaryFromReader(message: Resource, reader: jspb.BinaryReader): Resource;
}

export namespace Resource {
  export type AsObject = {
    memory: number,
    cpus: number,
    devicesList: Array<Device.AsObject>,
  }
}

export class Resources extends jspb.Message {
  hasReservations(): boolean;
  clearReservations(): void;
  getReservations(): Resource | undefined;
  setReservations(value?: Resource): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Resources.AsObject;
  static toObject(includeInstance: boolean, msg: Resources): Resources.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Resources, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Resources;
  static deserializeBinaryFromReader(message: Resources, reader: jspb.BinaryReader): Resources;
}

export namespace Resources {
  export type AsObject = {
    reservations?: Resource.AsObject,
  }
}

export class Deploy extends jspb.Message {
  getReplicas(): number;
  setReplicas(value: number): void;

  hasResources(): boolean;
  clearResources(): void;
  getResources(): Resources | undefined;
  setResources(value?: Resources): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Deploy.AsObject;
  static toObject(includeInstance: boolean, msg: Deploy): Deploy.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Deploy, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Deploy;
  static deserializeBinaryFromReader(message: Deploy, reader: jspb.BinaryReader): Deploy;
}

export namespace Deploy {
  export type AsObject = {
    replicas: number,
    resources?: Resources.AsObject,
  }
}

export class Port extends jspb.Message {
  getTarget(): number;
  setTarget(value: number): void;

  getProtocol(): ProtocolMap[keyof ProtocolMap];
  setProtocol(value: ProtocolMap[keyof ProtocolMap]): void;

  getMode(): ModeMap[keyof ModeMap];
  setMode(value: ModeMap[keyof ModeMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Port.AsObject;
  static toObject(includeInstance: boolean, msg: Port): Port.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Port, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Port;
  static deserializeBinaryFromReader(message: Port, reader: jspb.BinaryReader): Port;
}

export namespace Port {
  export type AsObject = {
    target: number,
    protocol: ProtocolMap[keyof ProtocolMap],
    mode: ModeMap[keyof ModeMap],
  }
}

export class Secret extends jspb.Message {
  getSource(): string;
  setSource(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Secret.AsObject;
  static toObject(includeInstance: boolean, msg: Secret): Secret.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Secret, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Secret;
  static deserializeBinaryFromReader(message: Secret, reader: jspb.BinaryReader): Secret;
}

export namespace Secret {
  export type AsObject = {
    source: string,
  }
}

export class Build extends jspb.Message {
  getContext(): string;
  setContext(value: string): void;

  getDockerfile(): string;
  setDockerfile(value: string): void;

  getArgsMap(): jspb.Map<string, string>;
  clearArgsMap(): void;
  getShmSize(): number;
  setShmSize(value: number): void;

  getTarget(): string;
  setTarget(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Build.AsObject;
  static toObject(includeInstance: boolean, msg: Build): Build.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Build, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Build;
  static deserializeBinaryFromReader(message: Build, reader: jspb.BinaryReader): Build;
}

export namespace Build {
  export type AsObject = {
    context: string,
    dockerfile: string,
    argsMap: Array<[string, string]>,
    shmSize: number,
    target: string,
  }
}

export class HealthCheck extends jspb.Message {
  clearTestList(): void;
  getTestList(): Array<string>;
  setTestList(value: Array<string>): void;
  addTest(value: string, index?: number): string;

  getInterval(): number;
  setInterval(value: number): void;

  getTimeout(): number;
  setTimeout(value: number): void;

  getRetries(): number;
  setRetries(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HealthCheck.AsObject;
  static toObject(includeInstance: boolean, msg: HealthCheck): HealthCheck.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: HealthCheck, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HealthCheck;
  static deserializeBinaryFromReader(message: HealthCheck, reader: jspb.BinaryReader): HealthCheck;
}

export namespace HealthCheck {
  export type AsObject = {
    testList: Array<string>,
    interval: number,
    timeout: number,
    retries: number,
  }
}

export class Service extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getImage(): string;
  setImage(value: string): void;

  getPlatform(): PlatformMap[keyof PlatformMap];
  setPlatform(value: PlatformMap[keyof PlatformMap]): void;

  getInternal(): boolean;
  setInternal(value: boolean): void;

  hasDeploy(): boolean;
  clearDeploy(): void;
  getDeploy(): Deploy | undefined;
  setDeploy(value?: Deploy): void;

  clearPortsList(): void;
  getPortsList(): Array<Port>;
  setPortsList(value: Array<Port>): void;
  addPorts(value?: Port, index?: number): Port;

  getEnvironmentMap(): jspb.Map<string, string>;
  clearEnvironmentMap(): void;
  hasBuild(): boolean;
  clearBuild(): void;
  getBuild(): Build | undefined;
  setBuild(value?: Build): void;

  clearSecretsList(): void;
  getSecretsList(): Array<Secret>;
  setSecretsList(value: Array<Secret>): void;
  addSecrets(value?: Secret, index?: number): Secret;

  hasHealthcheck(): boolean;
  clearHealthcheck(): void;
  getHealthcheck(): HealthCheck | undefined;
  setHealthcheck(value?: HealthCheck): void;

  clearCommandList(): void;
  getCommandList(): Array<string>;
  setCommandList(value: Array<string>): void;
  addCommand(value: string, index?: number): string;

  getDomainname(): string;
  setDomainname(value: string): void;

  getInit(): boolean;
  setInit(value: boolean): void;

  getDnsRole(): string;
  setDnsRole(value: string): void;

  hasStaticFiles(): boolean;
  clearStaticFiles(): void;
  getStaticFiles(): StaticFiles | undefined;
  setStaticFiles(value?: StaticFiles): void;

  getNetworks(): NetworkMap[keyof NetworkMap];
  setNetworks(value: NetworkMap[keyof NetworkMap]): void;

  hasRedis(): boolean;
  clearRedis(): void;
  getRedis(): Redis | undefined;
  setRedis(value?: Redis): void;

  hasPostgres(): boolean;
  clearPostgres(): void;
  getPostgres(): Postgres | undefined;
  setPostgres(value?: Postgres): void;

  getProject(): string;
  setProject(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Service.AsObject;
  static toObject(includeInstance: boolean, msg: Service): Service.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Service, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Service;
  static deserializeBinaryFromReader(message: Service, reader: jspb.BinaryReader): Service;
}

export namespace Service {
  export type AsObject = {
    name: string,
    image: string,
    platform: PlatformMap[keyof PlatformMap],
    internal: boolean,
    deploy?: Deploy.AsObject,
    portsList: Array<Port.AsObject>,
    environmentMap: Array<[string, string]>,
    build?: Build.AsObject,
    secretsList: Array<Secret.AsObject>,
    healthcheck?: HealthCheck.AsObject,
    commandList: Array<string>,
    domainname: string,
    init: boolean,
    dnsRole: string,
    staticFiles?: StaticFiles.AsObject,
    networks: NetworkMap[keyof NetworkMap],
    redis?: Redis.AsObject,
    postgres?: Postgres.AsObject,
    project: string,
  }
}

export class StaticFiles extends jspb.Message {
  getFolder(): string;
  setFolder(value: string): void;

  clearRedirectsList(): void;
  getRedirectsList(): Array<string>;
  setRedirectsList(value: Array<string>): void;
  addRedirects(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StaticFiles.AsObject;
  static toObject(includeInstance: boolean, msg: StaticFiles): StaticFiles.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StaticFiles, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StaticFiles;
  static deserializeBinaryFromReader(message: StaticFiles, reader: jspb.BinaryReader): StaticFiles;
}

export namespace StaticFiles {
  export type AsObject = {
    folder: string,
    redirectsList: Array<string>,
  }
}

export class Redis extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Redis.AsObject;
  static toObject(includeInstance: boolean, msg: Redis): Redis.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Redis, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Redis;
  static deserializeBinaryFromReader(message: Redis, reader: jspb.BinaryReader): Redis;
}

export namespace Redis {
  export type AsObject = {
  }
}

export class Postgres extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Postgres.AsObject;
  static toObject(includeInstance: boolean, msg: Postgres): Postgres.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Postgres, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Postgres;
  static deserializeBinaryFromReader(message: Postgres, reader: jspb.BinaryReader): Postgres;
}

export namespace Postgres {
  export type AsObject = {
  }
}

export class DeployEvent extends jspb.Message {
  getMode(): DeploymentModeMap[keyof DeploymentModeMap];
  setMode(value: DeploymentModeMap[keyof DeploymentModeMap]): void;

  getType(): string;
  setType(value: string): void;

  getSource(): string;
  setSource(value: string): void;

  getId(): string;
  setId(value: string): void;

  getDatacontenttype(): string;
  setDatacontenttype(value: string): void;

  getDataschema(): string;
  setDataschema(value: string): void;

  getSubject(): string;
  setSubject(value: string): void;

  hasTime(): boolean;
  clearTime(): void;
  getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getData(): Uint8Array | string;
  getData_asU8(): Uint8Array;
  getData_asB64(): string;
  setData(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeployEvent.AsObject;
  static toObject(includeInstance: boolean, msg: DeployEvent): DeployEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeployEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeployEvent;
  static deserializeBinaryFromReader(message: DeployEvent, reader: jspb.BinaryReader): DeployEvent;
}

export namespace DeployEvent {
  export type AsObject = {
    mode: DeploymentModeMap[keyof DeploymentModeMap],
    type: string,
    source: string,
    id: string,
    datacontenttype: string,
    dataschema: string,
    subject: string,
    time?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    data: Uint8Array | string,
  }
}

export class Event extends jspb.Message {
  getSpecversion(): string;
  setSpecversion(value: string): void;

  getType(): string;
  setType(value: string): void;

  getSource(): string;
  setSource(value: string): void;

  getId(): string;
  setId(value: string): void;

  getDatacontenttype(): string;
  setDatacontenttype(value: string): void;

  getDataschema(): string;
  setDataschema(value: string): void;

  getSubject(): string;
  setSubject(value: string): void;

  hasTime(): boolean;
  clearTime(): void;
  getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getData(): Uint8Array | string;
  getData_asU8(): Uint8Array;
  getData_asB64(): string;
  setData(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Event.AsObject;
  static toObject(includeInstance: boolean, msg: Event): Event.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Event, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Event;
  static deserializeBinaryFromReader(message: Event, reader: jspb.BinaryReader): Event;
}

export namespace Event {
  export type AsObject = {
    specversion: string,
    type: string,
    source: string,
    id: string,
    datacontenttype: string,
    dataschema: string,
    subject: string,
    time?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    data: Uint8Array | string,
  }
}

export class PublishRequest extends jspb.Message {
  hasEvent(): boolean;
  clearEvent(): void;
  getEvent(): Event | undefined;
  setEvent(value?: Event): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PublishRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PublishRequest): PublishRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PublishRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PublishRequest;
  static deserializeBinaryFromReader(message: PublishRequest, reader: jspb.BinaryReader): PublishRequest;
}

export namespace PublishRequest {
  export type AsObject = {
    event?: Event.AsObject,
  }
}

export class SubscribeRequest extends jspb.Message {
  clearServicesList(): void;
  getServicesList(): Array<string>;
  setServicesList(value: Array<string>): void;
  addServices(value: string, index?: number): string;

  getEtag(): string;
  setEtag(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SubscribeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SubscribeRequest): SubscribeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SubscribeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SubscribeRequest;
  static deserializeBinaryFromReader(message: SubscribeRequest, reader: jspb.BinaryReader): SubscribeRequest;
}

export namespace SubscribeRequest {
  export type AsObject = {
    servicesList: Array<string>,
    etag: string,
  }
}

export class SubscribeResponse extends jspb.Message {
  hasService(): boolean;
  clearService(): void;
  getService(): ServiceInfo | undefined;
  setService(value?: ServiceInfo): void;

  getName(): string;
  setName(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getState(): ServiceStateMap[keyof ServiceStateMap];
  setState(value: ServiceStateMap[keyof ServiceStateMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SubscribeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SubscribeResponse): SubscribeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SubscribeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SubscribeResponse;
  static deserializeBinaryFromReader(message: SubscribeResponse, reader: jspb.BinaryReader): SubscribeResponse;
}

export namespace SubscribeResponse {
  export type AsObject = {
    service?: ServiceInfo.AsObject,
    name: string,
    status: string,
    state: ServiceStateMap[keyof ServiceStateMap],
  }
}

export class DelegateSubdomainZoneRequest extends jspb.Message {
  clearNameServerRecordsList(): void;
  getNameServerRecordsList(): Array<string>;
  setNameServerRecordsList(value: Array<string>): void;
  addNameServerRecords(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DelegateSubdomainZoneRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DelegateSubdomainZoneRequest): DelegateSubdomainZoneRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DelegateSubdomainZoneRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DelegateSubdomainZoneRequest;
  static deserializeBinaryFromReader(message: DelegateSubdomainZoneRequest, reader: jspb.BinaryReader): DelegateSubdomainZoneRequest;
}

export namespace DelegateSubdomainZoneRequest {
  export type AsObject = {
    nameServerRecordsList: Array<string>,
  }
}

export class DelegateSubdomainZoneResponse extends jspb.Message {
  getZone(): string;
  setZone(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DelegateSubdomainZoneResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DelegateSubdomainZoneResponse): DelegateSubdomainZoneResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DelegateSubdomainZoneResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DelegateSubdomainZoneResponse;
  static deserializeBinaryFromReader(message: DelegateSubdomainZoneResponse, reader: jspb.BinaryReader): DelegateSubdomainZoneResponse;
}

export namespace DelegateSubdomainZoneResponse {
  export type AsObject = {
    zone: string,
  }
}

export class WhoAmIResponse extends jspb.Message {
  getTenant(): string;
  setTenant(value: string): void;

  getAccount(): string;
  setAccount(value: string): void;

  getRegion(): string;
  setRegion(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WhoAmIResponse.AsObject;
  static toObject(includeInstance: boolean, msg: WhoAmIResponse): WhoAmIResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: WhoAmIResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WhoAmIResponse;
  static deserializeBinaryFromReader(message: WhoAmIResponse, reader: jspb.BinaryReader): WhoAmIResponse;
}

export namespace WhoAmIResponse {
  export type AsObject = {
    tenant: string,
    account: string,
    region: string,
    userId: string,
  }
}

export interface DeploymentModeMap {
  UNSPECIFIED_MODE: 0;
  DEVELOPMENT: 1;
  STAGING: 2;
  PRODUCTION: 3;
}

export const DeploymentMode: DeploymentModeMap;

export interface ServiceStateMap {
  NOT_SPECIFIED: 0;
  BUILD_QUEUED: 1;
  BUILD_PROVISIONING: 2;
  BUILD_PENDING: 3;
  BUILD_ACTIVATING: 4;
  BUILD_RUNNING: 5;
  BUILD_STOPPING: 6;
  UPDATE_QUEUED: 7;
  DEPLOYMENT_PENDING: 8;
  DEPLOYMENT_COMPLETED: 9;
  DEPLOYMENT_FAILED: 10;
  BUILD_FAILED: 11;
}

export const ServiceState: ServiceStateMap;

export interface ConfigTypeMap {
  CONFIGTYPE_UNSPECIFIED: 0;
  CONFIGTYPE_SENSITIVE: 1;
}

export const ConfigType: ConfigTypeMap;

export interface PlatformMap {
  LINUX_AMD64: 0;
  LINUX_ARM64: 1;
  LINUX_ANY: 2;
}

export const Platform: PlatformMap;

export interface ProtocolMap {
  ANY: 0;
  UDP: 1;
  TCP: 2;
  HTTP: 3;
  HTTP2: 4;
  GRPC: 5;
}

export const Protocol: ProtocolMap;

export interface ModeMap {
  HOST: 0;
  INGRESS: 1;
}

export const Mode: ModeMap;

export interface NetworkMap {
  UNSPECIFIED: 0;
  PRIVATE: 1;
  PUBLIC: 2;
}

export const Network: NetworkMap;

