// package: io.defang.v1
// file: v1/fabric.proto

import * as jspb from "google-protobuf";

export class Void extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Void.AsObject;
  static toObject(includeInstance: boolean, msg: Void): Void.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Void, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Void;
  static deserializeBinaryFromReader(message: Void, reader: jspb.BinaryReader): Void;
}

export namespace Void {
  export type AsObject = {
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

export class LogEntry extends jspb.Message {
  getMessage(): string;
  setMessage(value: string): void;

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
  }
}

export class Services extends jspb.Message {
  clearServicesList(): void;
  getServicesList(): Array<Service>;
  setServicesList(value: Array<Service>): void;
  addServices(value?: Service, index?: number): Service;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Services.AsObject;
  static toObject(includeInstance: boolean, msg: Services): Services.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Services, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Services;
  static deserializeBinaryFromReader(message: Services, reader: jspb.BinaryReader): Services;
}

export namespace Services {
  export type AsObject = {
    servicesList: Array<Service.AsObject>,
  }
}

export class Resource extends jspb.Message {
  getMemory(): number;
  setMemory(value: number): void;

  getCpus(): number;
  setCpus(value: number): void;

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
  }
}

export class Build extends jspb.Message {
  getContext(): string;
  setContext(value: string): void;

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
  }
}

export class Timestamp extends jspb.Message {
  getSeconds(): number;
  setSeconds(value: number): void;

  getNanos(): number;
  setNanos(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Timestamp.AsObject;
  static toObject(includeInstance: boolean, msg: Timestamp): Timestamp.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Timestamp, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Timestamp;
  static deserializeBinaryFromReader(message: Timestamp, reader: jspb.BinaryReader): Timestamp;
}

export namespace Timestamp {
  export type AsObject = {
    seconds: number,
    nanos: number,
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
  getTime(): Timestamp | undefined;
  setTime(value?: Timestamp): void;

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
    time?: Timestamp.AsObject,
    data: Uint8Array | string,
  }
}

export interface PlatformMap {
  LINUX_AMD64: 0;
  LINUX_ARM64: 1;
  LINUX_ANY: 2;
}

export const Platform: PlatformMap;

export interface ProtocolMap {
  TCP: 0;
  UDP: 1;
}

export const Protocol: ProtocolMap;
