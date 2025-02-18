// source: io/defang/v1/fabric.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global = (function() {
  if (this) { return this; }
  if (typeof window !== 'undefined') { return window; }
  if (typeof global !== 'undefined') { return global; }
  if (typeof self !== 'undefined') { return self; }
  return Function('return this')();
}.call(null));

var google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js');
goog.object.extend(proto, google_protobuf_empty_pb);
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');
goog.object.extend(proto, google_protobuf_timestamp_pb);
goog.exportSymbol('proto.io.defang.v1.Build', null, global);
goog.exportSymbol('proto.io.defang.v1.CanIUseRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.CanIUseResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.CodeChange', null, global);
goog.exportSymbol('proto.io.defang.v1.Config', null, global);
goog.exportSymbol('proto.io.defang.v1.ConfigKey', null, global);
goog.exportSymbol('proto.io.defang.v1.ConfigType', null, global);
goog.exportSymbol('proto.io.defang.v1.DebugRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.DebugResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.DelegateSubdomainZoneRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.DelegateSubdomainZoneResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.DeleteConfigsRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.DeleteRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.DeleteResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.Deploy', null, global);
goog.exportSymbol('proto.io.defang.v1.DeployEvent', null, global);
goog.exportSymbol('proto.io.defang.v1.DeployRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.DeployResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.Deployment', null, global);
goog.exportSymbol('proto.io.defang.v1.DeploymentAction', null, global);
goog.exportSymbol('proto.io.defang.v1.DeploymentMode', null, global);
goog.exportSymbol('proto.io.defang.v1.DestroyRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.DestroyResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.Device', null, global);
goog.exportSymbol('proto.io.defang.v1.Event', null, global);
goog.exportSymbol('proto.io.defang.v1.File', null, global);
goog.exportSymbol('proto.io.defang.v1.GenerateFilesRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.GenerateFilesResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.GenerateStatusRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.GetConfigsRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.GetConfigsResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.GetRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.GetSelectedProviderRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.GetSelectedProviderResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.GetServicesRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.GetServicesResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.HealthCheck', null, global);
goog.exportSymbol('proto.io.defang.v1.Issue', null, global);
goog.exportSymbol('proto.io.defang.v1.ListConfigsRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.ListConfigsResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.ListDeploymentsRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.ListDeploymentsResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.LogEntry', null, global);
goog.exportSymbol('proto.io.defang.v1.Mode', null, global);
goog.exportSymbol('proto.io.defang.v1.Network', null, global);
goog.exportSymbol('proto.io.defang.v1.Platform', null, global);
goog.exportSymbol('proto.io.defang.v1.Port', null, global);
goog.exportSymbol('proto.io.defang.v1.ProjectUpdate', null, global);
goog.exportSymbol('proto.io.defang.v1.Protocol', null, global);
goog.exportSymbol('proto.io.defang.v1.Provider', null, global);
goog.exportSymbol('proto.io.defang.v1.PublishRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.PutConfigRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.PutDeploymentRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.Redis', null, global);
goog.exportSymbol('proto.io.defang.v1.Resource', null, global);
goog.exportSymbol('proto.io.defang.v1.Resources', null, global);
goog.exportSymbol('proto.io.defang.v1.Secret', null, global);
goog.exportSymbol('proto.io.defang.v1.SecretValue', null, global);
goog.exportSymbol('proto.io.defang.v1.Secrets', null, global);
goog.exportSymbol('proto.io.defang.v1.Service', null, global);
goog.exportSymbol('proto.io.defang.v1.ServiceInfo', null, global);
goog.exportSymbol('proto.io.defang.v1.ServiceState', null, global);
goog.exportSymbol('proto.io.defang.v1.SetOptionsRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.SetSelectedProviderRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.StartGenerateResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.StaticFiles', null, global);
goog.exportSymbol('proto.io.defang.v1.Status', null, global);
goog.exportSymbol('proto.io.defang.v1.SubscribeRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.SubscribeResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.SubscriptionTier', null, global);
goog.exportSymbol('proto.io.defang.v1.TailRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.TailResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.TokenRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.TokenResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.TrackRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.UploadURLRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.UploadURLResponse', null, global);
goog.exportSymbol('proto.io.defang.v1.VerifyDNSSetupRequest', null, global);
goog.exportSymbol('proto.io.defang.v1.Version', null, global);
goog.exportSymbol('proto.io.defang.v1.WhoAmIResponse', null, global);
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GetSelectedProviderRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.GetSelectedProviderRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GetSelectedProviderRequest.displayName = 'proto.io.defang.v1.GetSelectedProviderRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GetSelectedProviderResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.GetSelectedProviderResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GetSelectedProviderResponse.displayName = 'proto.io.defang.v1.GetSelectedProviderResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.SetSelectedProviderRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.SetSelectedProviderRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.SetSelectedProviderRequest.displayName = 'proto.io.defang.v1.SetSelectedProviderRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.VerifyDNSSetupRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.VerifyDNSSetupRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.VerifyDNSSetupRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.VerifyDNSSetupRequest.displayName = 'proto.io.defang.v1.VerifyDNSSetupRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DestroyRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.DestroyRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DestroyRequest.displayName = 'proto.io.defang.v1.DestroyRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DestroyResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.DestroyResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DestroyResponse.displayName = 'proto.io.defang.v1.DestroyResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DebugRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.DebugRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.DebugRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DebugRequest.displayName = 'proto.io.defang.v1.DebugRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DebugResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.DebugResponse.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.DebugResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DebugResponse.displayName = 'proto.io.defang.v1.DebugResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Issue = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.Issue.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.Issue, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Issue.displayName = 'proto.io.defang.v1.Issue';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.CodeChange = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.CodeChange, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.CodeChange.displayName = 'proto.io.defang.v1.CodeChange';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.TrackRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.TrackRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.TrackRequest.displayName = 'proto.io.defang.v1.TrackRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.CanIUseRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.CanIUseRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.CanIUseRequest.displayName = 'proto.io.defang.v1.CanIUseRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.CanIUseResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.CanIUseResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.CanIUseResponse.displayName = 'proto.io.defang.v1.CanIUseResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DeployRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.DeployRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.DeployRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DeployRequest.displayName = 'proto.io.defang.v1.DeployRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DeployResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.DeployResponse.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.DeployResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DeployResponse.displayName = 'proto.io.defang.v1.DeployResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DeleteRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.DeleteRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.DeleteRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DeleteRequest.displayName = 'proto.io.defang.v1.DeleteRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DeleteResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.DeleteResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DeleteResponse.displayName = 'proto.io.defang.v1.DeleteResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GenerateFilesRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.GenerateFilesRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GenerateFilesRequest.displayName = 'proto.io.defang.v1.GenerateFilesRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.File = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.File, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.File.displayName = 'proto.io.defang.v1.File';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GenerateFilesResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.GenerateFilesResponse.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.GenerateFilesResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GenerateFilesResponse.displayName = 'proto.io.defang.v1.GenerateFilesResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.StartGenerateResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.StartGenerateResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.StartGenerateResponse.displayName = 'proto.io.defang.v1.StartGenerateResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GenerateStatusRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.GenerateStatusRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GenerateStatusRequest.displayName = 'proto.io.defang.v1.GenerateStatusRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.UploadURLRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.UploadURLRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.UploadURLRequest.displayName = 'proto.io.defang.v1.UploadURLRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.UploadURLResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.UploadURLResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.UploadURLResponse.displayName = 'proto.io.defang.v1.UploadURLResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.ServiceInfo = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.ServiceInfo.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.ServiceInfo, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.ServiceInfo.displayName = 'proto.io.defang.v1.ServiceInfo';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Secrets = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.Secrets.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.Secrets, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Secrets.displayName = 'proto.io.defang.v1.Secrets';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.SecretValue = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.SecretValue, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.SecretValue.displayName = 'proto.io.defang.v1.SecretValue';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Config = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Config, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Config.displayName = 'proto.io.defang.v1.Config';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.ConfigKey = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.ConfigKey, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.ConfigKey.displayName = 'proto.io.defang.v1.ConfigKey';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.PutConfigRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.PutConfigRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.PutConfigRequest.displayName = 'proto.io.defang.v1.PutConfigRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GetConfigsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.GetConfigsRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.GetConfigsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GetConfigsRequest.displayName = 'proto.io.defang.v1.GetConfigsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GetConfigsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.GetConfigsResponse.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.GetConfigsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GetConfigsResponse.displayName = 'proto.io.defang.v1.GetConfigsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DeleteConfigsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.DeleteConfigsRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.DeleteConfigsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DeleteConfigsRequest.displayName = 'proto.io.defang.v1.DeleteConfigsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.ListConfigsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.ListConfigsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.ListConfigsRequest.displayName = 'proto.io.defang.v1.ListConfigsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.ListConfigsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.ListConfigsResponse.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.ListConfigsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.ListConfigsResponse.displayName = 'proto.io.defang.v1.ListConfigsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Deployment = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Deployment, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Deployment.displayName = 'proto.io.defang.v1.Deployment';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.PutDeploymentRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.PutDeploymentRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.PutDeploymentRequest.displayName = 'proto.io.defang.v1.PutDeploymentRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.ListDeploymentsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.ListDeploymentsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.ListDeploymentsRequest.displayName = 'proto.io.defang.v1.ListDeploymentsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.ListDeploymentsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.ListDeploymentsResponse.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.ListDeploymentsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.ListDeploymentsResponse.displayName = 'proto.io.defang.v1.ListDeploymentsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.TokenRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.TokenRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.TokenRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.TokenRequest.displayName = 'proto.io.defang.v1.TokenRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.TokenResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.TokenResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.TokenResponse.displayName = 'proto.io.defang.v1.TokenResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Status = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Status, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Status.displayName = 'proto.io.defang.v1.Status';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Version = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Version, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Version.displayName = 'proto.io.defang.v1.Version';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.TailRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.TailRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.TailRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.TailRequest.displayName = 'proto.io.defang.v1.TailRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.LogEntry = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.LogEntry, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.LogEntry.displayName = 'proto.io.defang.v1.LogEntry';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.TailResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.TailResponse.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.TailResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.TailResponse.displayName = 'proto.io.defang.v1.TailResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GetServicesResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.GetServicesResponse.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.GetServicesResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GetServicesResponse.displayName = 'proto.io.defang.v1.GetServicesResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.ProjectUpdate = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.ProjectUpdate.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.ProjectUpdate, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.ProjectUpdate.displayName = 'proto.io.defang.v1.ProjectUpdate';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GetRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.GetRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GetRequest.displayName = 'proto.io.defang.v1.GetRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Device = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.Device.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.Device, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Device.displayName = 'proto.io.defang.v1.Device';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Resource = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.Resource.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.Resource, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Resource.displayName = 'proto.io.defang.v1.Resource';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Resources = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Resources, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Resources.displayName = 'proto.io.defang.v1.Resources';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Deploy = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Deploy, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Deploy.displayName = 'proto.io.defang.v1.Deploy';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Port = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Port, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Port.displayName = 'proto.io.defang.v1.Port';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Secret = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Secret, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Secret.displayName = 'proto.io.defang.v1.Secret';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Build = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Build, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Build.displayName = 'proto.io.defang.v1.Build';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.HealthCheck = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.HealthCheck.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.HealthCheck, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.HealthCheck.displayName = 'proto.io.defang.v1.HealthCheck';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Service = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.Service.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.Service, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Service.displayName = 'proto.io.defang.v1.Service';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.StaticFiles = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.StaticFiles.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.StaticFiles, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.StaticFiles.displayName = 'proto.io.defang.v1.StaticFiles';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Redis = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Redis, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Redis.displayName = 'proto.io.defang.v1.Redis';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DeployEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.DeployEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DeployEvent.displayName = 'proto.io.defang.v1.DeployEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.Event = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.Event, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.Event.displayName = 'proto.io.defang.v1.Event';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.PublishRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.PublishRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.PublishRequest.displayName = 'proto.io.defang.v1.PublishRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.SubscribeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.SubscribeRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.SubscribeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.SubscribeRequest.displayName = 'proto.io.defang.v1.SubscribeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.SubscribeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.SubscribeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.SubscribeResponse.displayName = 'proto.io.defang.v1.SubscribeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.GetServicesRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.GetServicesRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.GetServicesRequest.displayName = 'proto.io.defang.v1.GetServicesRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.io.defang.v1.DelegateSubdomainZoneRequest.repeatedFields_, null);
};
goog.inherits(proto.io.defang.v1.DelegateSubdomainZoneRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DelegateSubdomainZoneRequest.displayName = 'proto.io.defang.v1.DelegateSubdomainZoneRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.DelegateSubdomainZoneResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.DelegateSubdomainZoneResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.DelegateSubdomainZoneResponse.displayName = 'proto.io.defang.v1.DelegateSubdomainZoneResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.SetOptionsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.SetOptionsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.SetOptionsRequest.displayName = 'proto.io.defang.v1.SetOptionsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.io.defang.v1.WhoAmIResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.io.defang.v1.WhoAmIResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.io.defang.v1.WhoAmIResponse.displayName = 'proto.io.defang.v1.WhoAmIResponse';
}



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GetSelectedProviderRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GetSelectedProviderRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GetSelectedProviderRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetSelectedProviderRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    project: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GetSelectedProviderRequest}
 */
proto.io.defang.v1.GetSelectedProviderRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GetSelectedProviderRequest;
  return proto.io.defang.v1.GetSelectedProviderRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GetSelectedProviderRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GetSelectedProviderRequest}
 */
proto.io.defang.v1.GetSelectedProviderRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GetSelectedProviderRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GetSelectedProviderRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GetSelectedProviderRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetSelectedProviderRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string project = 1;
 * @return {string}
 */
proto.io.defang.v1.GetSelectedProviderRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.GetSelectedProviderRequest} returns this
 */
proto.io.defang.v1.GetSelectedProviderRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GetSelectedProviderResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GetSelectedProviderResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GetSelectedProviderResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetSelectedProviderResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    provider: jspb.Message.getFieldWithDefault(msg, 1, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GetSelectedProviderResponse}
 */
proto.io.defang.v1.GetSelectedProviderResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GetSelectedProviderResponse;
  return proto.io.defang.v1.GetSelectedProviderResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GetSelectedProviderResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GetSelectedProviderResponse}
 */
proto.io.defang.v1.GetSelectedProviderResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.io.defang.v1.Provider} */ (reader.readEnum());
      msg.setProvider(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GetSelectedProviderResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GetSelectedProviderResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GetSelectedProviderResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetSelectedProviderResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProvider();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
};


/**
 * optional Provider provider = 1;
 * @return {!proto.io.defang.v1.Provider}
 */
proto.io.defang.v1.GetSelectedProviderResponse.prototype.getProvider = function() {
  return /** @type {!proto.io.defang.v1.Provider} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.io.defang.v1.Provider} value
 * @return {!proto.io.defang.v1.GetSelectedProviderResponse} returns this
 */
proto.io.defang.v1.GetSelectedProviderResponse.prototype.setProvider = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.SetSelectedProviderRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.SetSelectedProviderRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.SetSelectedProviderRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SetSelectedProviderRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    project: jspb.Message.getFieldWithDefault(msg, 1, ""),
    provider: jspb.Message.getFieldWithDefault(msg, 2, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.SetSelectedProviderRequest}
 */
proto.io.defang.v1.SetSelectedProviderRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.SetSelectedProviderRequest;
  return proto.io.defang.v1.SetSelectedProviderRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.SetSelectedProviderRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.SetSelectedProviderRequest}
 */
proto.io.defang.v1.SetSelectedProviderRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 2:
      var value = /** @type {!proto.io.defang.v1.Provider} */ (reader.readEnum());
      msg.setProvider(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.SetSelectedProviderRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.SetSelectedProviderRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.SetSelectedProviderRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SetSelectedProviderRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getProvider();
  if (f !== 0.0) {
    writer.writeEnum(
      2,
      f
    );
  }
};


/**
 * optional string project = 1;
 * @return {string}
 */
proto.io.defang.v1.SetSelectedProviderRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.SetSelectedProviderRequest} returns this
 */
proto.io.defang.v1.SetSelectedProviderRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional Provider provider = 2;
 * @return {!proto.io.defang.v1.Provider}
 */
proto.io.defang.v1.SetSelectedProviderRequest.prototype.getProvider = function() {
  return /** @type {!proto.io.defang.v1.Provider} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {!proto.io.defang.v1.Provider} value
 * @return {!proto.io.defang.v1.SetSelectedProviderRequest} returns this
 */
proto.io.defang.v1.SetSelectedProviderRequest.prototype.setProvider = function(value) {
  return jspb.Message.setProto3EnumField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.VerifyDNSSetupRequest.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.VerifyDNSSetupRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.VerifyDNSSetupRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.VerifyDNSSetupRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.VerifyDNSSetupRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    domain: jspb.Message.getFieldWithDefault(msg, 1, ""),
    targetsList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.VerifyDNSSetupRequest}
 */
proto.io.defang.v1.VerifyDNSSetupRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.VerifyDNSSetupRequest;
  return proto.io.defang.v1.VerifyDNSSetupRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.VerifyDNSSetupRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.VerifyDNSSetupRequest}
 */
proto.io.defang.v1.VerifyDNSSetupRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setDomain(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.addTargets(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.VerifyDNSSetupRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.VerifyDNSSetupRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.VerifyDNSSetupRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.VerifyDNSSetupRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDomain();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getTargetsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      2,
      f
    );
  }
};


/**
 * optional string domain = 1;
 * @return {string}
 */
proto.io.defang.v1.VerifyDNSSetupRequest.prototype.getDomain = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.VerifyDNSSetupRequest} returns this
 */
proto.io.defang.v1.VerifyDNSSetupRequest.prototype.setDomain = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated string targets = 2;
 * @return {!Array<string>}
 */
proto.io.defang.v1.VerifyDNSSetupRequest.prototype.getTargetsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.VerifyDNSSetupRequest} returns this
 */
proto.io.defang.v1.VerifyDNSSetupRequest.prototype.setTargetsList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.VerifyDNSSetupRequest} returns this
 */
proto.io.defang.v1.VerifyDNSSetupRequest.prototype.addTargets = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.VerifyDNSSetupRequest} returns this
 */
proto.io.defang.v1.VerifyDNSSetupRequest.prototype.clearTargetsList = function() {
  return this.setTargetsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DestroyRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DestroyRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DestroyRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DestroyRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    project: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DestroyRequest}
 */
proto.io.defang.v1.DestroyRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DestroyRequest;
  return proto.io.defang.v1.DestroyRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DestroyRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DestroyRequest}
 */
proto.io.defang.v1.DestroyRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DestroyRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DestroyRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DestroyRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DestroyRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string project = 1;
 * @return {string}
 */
proto.io.defang.v1.DestroyRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DestroyRequest} returns this
 */
proto.io.defang.v1.DestroyRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DestroyResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DestroyResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DestroyResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DestroyResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    etag: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DestroyResponse}
 */
proto.io.defang.v1.DestroyResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DestroyResponse;
  return proto.io.defang.v1.DestroyResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DestroyResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DestroyResponse}
 */
proto.io.defang.v1.DestroyResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setEtag(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DestroyResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DestroyResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DestroyResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DestroyResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEtag();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string etag = 1;
 * @return {string}
 */
proto.io.defang.v1.DestroyResponse.prototype.getEtag = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DestroyResponse} returns this
 */
proto.io.defang.v1.DestroyResponse.prototype.setEtag = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.DebugRequest.repeatedFields_ = [1,5];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DebugRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DebugRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DebugRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DebugRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    filesList: jspb.Message.toObjectList(msg.getFilesList(),
    proto.io.defang.v1.File.toObject, includeInstance),
    etag: jspb.Message.getFieldWithDefault(msg, 2, ""),
    project: jspb.Message.getFieldWithDefault(msg, 3, ""),
    logs: jspb.Message.getFieldWithDefault(msg, 4, ""),
    servicesList: (f = jspb.Message.getRepeatedField(msg, 5)) == null ? undefined : f,
    trainingOptOut: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    since: (f = msg.getSince()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DebugRequest}
 */
proto.io.defang.v1.DebugRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DebugRequest;
  return proto.io.defang.v1.DebugRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DebugRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DebugRequest}
 */
proto.io.defang.v1.DebugRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.File;
      reader.readMessage(value,proto.io.defang.v1.File.deserializeBinaryFromReader);
      msg.addFiles(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setEtag(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setLogs(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.addServices(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setTrainingOptOut(value);
      break;
    case 7:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setSince(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DebugRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DebugRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DebugRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DebugRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getFilesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.File.serializeBinaryToWriter
    );
  }
  f = message.getEtag();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getLogs();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getServicesList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      5,
      f
    );
  }
  f = message.getTrainingOptOut();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getSince();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * repeated File files = 1;
 * @return {!Array<!proto.io.defang.v1.File>}
 */
proto.io.defang.v1.DebugRequest.prototype.getFilesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.File>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.File, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.File>} value
 * @return {!proto.io.defang.v1.DebugRequest} returns this
*/
proto.io.defang.v1.DebugRequest.prototype.setFilesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.File=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.File}
 */
proto.io.defang.v1.DebugRequest.prototype.addFiles = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.File, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.DebugRequest} returns this
 */
proto.io.defang.v1.DebugRequest.prototype.clearFilesList = function() {
  return this.setFilesList([]);
};


/**
 * optional string etag = 2;
 * @return {string}
 */
proto.io.defang.v1.DebugRequest.prototype.getEtag = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DebugRequest} returns this
 */
proto.io.defang.v1.DebugRequest.prototype.setEtag = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string project = 3;
 * @return {string}
 */
proto.io.defang.v1.DebugRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DebugRequest} returns this
 */
proto.io.defang.v1.DebugRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string logs = 4;
 * @return {string}
 */
proto.io.defang.v1.DebugRequest.prototype.getLogs = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DebugRequest} returns this
 */
proto.io.defang.v1.DebugRequest.prototype.setLogs = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * repeated string services = 5;
 * @return {!Array<string>}
 */
proto.io.defang.v1.DebugRequest.prototype.getServicesList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 5));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.DebugRequest} returns this
 */
proto.io.defang.v1.DebugRequest.prototype.setServicesList = function(value) {
  return jspb.Message.setField(this, 5, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.DebugRequest} returns this
 */
proto.io.defang.v1.DebugRequest.prototype.addServices = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 5, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.DebugRequest} returns this
 */
proto.io.defang.v1.DebugRequest.prototype.clearServicesList = function() {
  return this.setServicesList([]);
};


/**
 * optional bool training_opt_out = 6;
 * @return {boolean}
 */
proto.io.defang.v1.DebugRequest.prototype.getTrainingOptOut = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.DebugRequest} returns this
 */
proto.io.defang.v1.DebugRequest.prototype.setTrainingOptOut = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional google.protobuf.Timestamp since = 7;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.io.defang.v1.DebugRequest.prototype.getSince = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 7));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.io.defang.v1.DebugRequest} returns this
*/
proto.io.defang.v1.DebugRequest.prototype.setSince = function(value) {
  return jspb.Message.setWrapperField(this, 7, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.DebugRequest} returns this
 */
proto.io.defang.v1.DebugRequest.prototype.clearSince = function() {
  return this.setSince(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.DebugRequest.prototype.hasSince = function() {
  return jspb.Message.getField(this, 7) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.DebugResponse.repeatedFields_ = [2,3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DebugResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DebugResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DebugResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DebugResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    general: jspb.Message.getFieldWithDefault(msg, 1, ""),
    issuesList: jspb.Message.toObjectList(msg.getIssuesList(),
    proto.io.defang.v1.Issue.toObject, includeInstance),
    requestsList: (f = jspb.Message.getRepeatedField(msg, 3)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DebugResponse}
 */
proto.io.defang.v1.DebugResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DebugResponse;
  return proto.io.defang.v1.DebugResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DebugResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DebugResponse}
 */
proto.io.defang.v1.DebugResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setGeneral(value);
      break;
    case 2:
      var value = new proto.io.defang.v1.Issue;
      reader.readMessage(value,proto.io.defang.v1.Issue.deserializeBinaryFromReader);
      msg.addIssues(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.addRequests(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DebugResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DebugResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DebugResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DebugResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGeneral();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getIssuesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      proto.io.defang.v1.Issue.serializeBinaryToWriter
    );
  }
  f = message.getRequestsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      3,
      f
    );
  }
};


/**
 * optional string general = 1;
 * @return {string}
 */
proto.io.defang.v1.DebugResponse.prototype.getGeneral = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DebugResponse} returns this
 */
proto.io.defang.v1.DebugResponse.prototype.setGeneral = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated Issue issues = 2;
 * @return {!Array<!proto.io.defang.v1.Issue>}
 */
proto.io.defang.v1.DebugResponse.prototype.getIssuesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.Issue>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.Issue, 2));
};


/**
 * @param {!Array<!proto.io.defang.v1.Issue>} value
 * @return {!proto.io.defang.v1.DebugResponse} returns this
*/
proto.io.defang.v1.DebugResponse.prototype.setIssuesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.io.defang.v1.Issue=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Issue}
 */
proto.io.defang.v1.DebugResponse.prototype.addIssues = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.io.defang.v1.Issue, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.DebugResponse} returns this
 */
proto.io.defang.v1.DebugResponse.prototype.clearIssuesList = function() {
  return this.setIssuesList([]);
};


/**
 * repeated string requests = 3;
 * @return {!Array<string>}
 */
proto.io.defang.v1.DebugResponse.prototype.getRequestsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 3));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.DebugResponse} returns this
 */
proto.io.defang.v1.DebugResponse.prototype.setRequestsList = function(value) {
  return jspb.Message.setField(this, 3, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.DebugResponse} returns this
 */
proto.io.defang.v1.DebugResponse.prototype.addRequests = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 3, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.DebugResponse} returns this
 */
proto.io.defang.v1.DebugResponse.prototype.clearRequestsList = function() {
  return this.setRequestsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.Issue.repeatedFields_ = [4];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Issue.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Issue.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Issue} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Issue.toObject = function(includeInstance, msg) {
  var f, obj = {
    type: jspb.Message.getFieldWithDefault(msg, 1, ""),
    severity: jspb.Message.getFieldWithDefault(msg, 2, ""),
    details: jspb.Message.getFieldWithDefault(msg, 3, ""),
    codeChangesList: jspb.Message.toObjectList(msg.getCodeChangesList(),
    proto.io.defang.v1.CodeChange.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Issue}
 */
proto.io.defang.v1.Issue.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Issue;
  return proto.io.defang.v1.Issue.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Issue} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Issue}
 */
proto.io.defang.v1.Issue.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setType(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setSeverity(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setDetails(value);
      break;
    case 4:
      var value = new proto.io.defang.v1.CodeChange;
      reader.readMessage(value,proto.io.defang.v1.CodeChange.deserializeBinaryFromReader);
      msg.addCodeChanges(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Issue.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Issue.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Issue} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Issue.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getType();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getSeverity();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getDetails();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getCodeChangesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      4,
      f,
      proto.io.defang.v1.CodeChange.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type = 1;
 * @return {string}
 */
proto.io.defang.v1.Issue.prototype.getType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Issue} returns this
 */
proto.io.defang.v1.Issue.prototype.setType = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string severity = 2;
 * @return {string}
 */
proto.io.defang.v1.Issue.prototype.getSeverity = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Issue} returns this
 */
proto.io.defang.v1.Issue.prototype.setSeverity = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string details = 3;
 * @return {string}
 */
proto.io.defang.v1.Issue.prototype.getDetails = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Issue} returns this
 */
proto.io.defang.v1.Issue.prototype.setDetails = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * repeated CodeChange code_changes = 4;
 * @return {!Array<!proto.io.defang.v1.CodeChange>}
 */
proto.io.defang.v1.Issue.prototype.getCodeChangesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.CodeChange>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.CodeChange, 4));
};


/**
 * @param {!Array<!proto.io.defang.v1.CodeChange>} value
 * @return {!proto.io.defang.v1.Issue} returns this
*/
proto.io.defang.v1.Issue.prototype.setCodeChangesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 4, value);
};


/**
 * @param {!proto.io.defang.v1.CodeChange=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.CodeChange}
 */
proto.io.defang.v1.Issue.prototype.addCodeChanges = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 4, opt_value, proto.io.defang.v1.CodeChange, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.Issue} returns this
 */
proto.io.defang.v1.Issue.prototype.clearCodeChangesList = function() {
  return this.setCodeChangesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.CodeChange.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.CodeChange.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.CodeChange} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.CodeChange.toObject = function(includeInstance, msg) {
  var f, obj = {
    file: jspb.Message.getFieldWithDefault(msg, 1, ""),
    change: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.CodeChange}
 */
proto.io.defang.v1.CodeChange.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.CodeChange;
  return proto.io.defang.v1.CodeChange.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.CodeChange} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.CodeChange}
 */
proto.io.defang.v1.CodeChange.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setFile(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setChange(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.CodeChange.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.CodeChange.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.CodeChange} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.CodeChange.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getFile();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getChange();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string file = 1;
 * @return {string}
 */
proto.io.defang.v1.CodeChange.prototype.getFile = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.CodeChange} returns this
 */
proto.io.defang.v1.CodeChange.prototype.setFile = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string change = 2;
 * @return {string}
 */
proto.io.defang.v1.CodeChange.prototype.getChange = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.CodeChange} returns this
 */
proto.io.defang.v1.CodeChange.prototype.setChange = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.TrackRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.TrackRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.TrackRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TrackRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    anonId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    event: jspb.Message.getFieldWithDefault(msg, 2, ""),
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, undefined) : [],
    os: jspb.Message.getFieldWithDefault(msg, 4, ""),
    arch: jspb.Message.getFieldWithDefault(msg, 5, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.TrackRequest}
 */
proto.io.defang.v1.TrackRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.TrackRequest;
  return proto.io.defang.v1.TrackRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.TrackRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.TrackRequest}
 */
proto.io.defang.v1.TrackRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setAnonId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setEvent(value);
      break;
    case 3:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readString, null, "", "");
         });
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setOs(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setArch(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.TrackRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.TrackRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.TrackRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TrackRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getAnonId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getEvent();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(3, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeString);
  }
  f = message.getOs();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getArch();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
};


/**
 * optional string anon_id = 1;
 * @return {string}
 */
proto.io.defang.v1.TrackRequest.prototype.getAnonId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TrackRequest} returns this
 */
proto.io.defang.v1.TrackRequest.prototype.setAnonId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string event = 2;
 * @return {string}
 */
proto.io.defang.v1.TrackRequest.prototype.getEvent = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TrackRequest} returns this
 */
proto.io.defang.v1.TrackRequest.prototype.setEvent = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * map<string, string> properties = 3;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,string>}
 */
proto.io.defang.v1.TrackRequest.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,string>} */ (
      jspb.Message.getMapField(this, 3, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.io.defang.v1.TrackRequest} returns this
 */
proto.io.defang.v1.TrackRequest.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * optional string os = 4;
 * @return {string}
 */
proto.io.defang.v1.TrackRequest.prototype.getOs = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TrackRequest} returns this
 */
proto.io.defang.v1.TrackRequest.prototype.setOs = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string arch = 5;
 * @return {string}
 */
proto.io.defang.v1.TrackRequest.prototype.getArch = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TrackRequest} returns this
 */
proto.io.defang.v1.TrackRequest.prototype.setArch = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.CanIUseRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.CanIUseRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.CanIUseRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.CanIUseRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    project: jspb.Message.getFieldWithDefault(msg, 1, ""),
    provider: jspb.Message.getFieldWithDefault(msg, 2, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.CanIUseRequest}
 */
proto.io.defang.v1.CanIUseRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.CanIUseRequest;
  return proto.io.defang.v1.CanIUseRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.CanIUseRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.CanIUseRequest}
 */
proto.io.defang.v1.CanIUseRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 2:
      var value = /** @type {!proto.io.defang.v1.Provider} */ (reader.readEnum());
      msg.setProvider(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.CanIUseRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.CanIUseRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.CanIUseRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.CanIUseRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getProvider();
  if (f !== 0.0) {
    writer.writeEnum(
      2,
      f
    );
  }
};


/**
 * optional string project = 1;
 * @return {string}
 */
proto.io.defang.v1.CanIUseRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.CanIUseRequest} returns this
 */
proto.io.defang.v1.CanIUseRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional Provider provider = 2;
 * @return {!proto.io.defang.v1.Provider}
 */
proto.io.defang.v1.CanIUseRequest.prototype.getProvider = function() {
  return /** @type {!proto.io.defang.v1.Provider} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {!proto.io.defang.v1.Provider} value
 * @return {!proto.io.defang.v1.CanIUseRequest} returns this
 */
proto.io.defang.v1.CanIUseRequest.prototype.setProvider = function(value) {
  return jspb.Message.setProto3EnumField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.CanIUseResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.CanIUseResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.CanIUseResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.CanIUseResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    cdImage: jspb.Message.getFieldWithDefault(msg, 2, ""),
    gpu: jspb.Message.getBooleanFieldWithDefault(msg, 3, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.CanIUseResponse}
 */
proto.io.defang.v1.CanIUseResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.CanIUseResponse;
  return proto.io.defang.v1.CanIUseResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.CanIUseResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.CanIUseResponse}
 */
proto.io.defang.v1.CanIUseResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setCdImage(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setGpu(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.CanIUseResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.CanIUseResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.CanIUseResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.CanIUseResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCdImage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getGpu();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
};


/**
 * optional string cd_image = 2;
 * @return {string}
 */
proto.io.defang.v1.CanIUseResponse.prototype.getCdImage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.CanIUseResponse} returns this
 */
proto.io.defang.v1.CanIUseResponse.prototype.setCdImage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional bool gpu = 3;
 * @return {boolean}
 */
proto.io.defang.v1.CanIUseResponse.prototype.getGpu = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.CanIUseResponse} returns this
 */
proto.io.defang.v1.CanIUseResponse.prototype.setGpu = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.DeployRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DeployRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DeployRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DeployRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeployRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    servicesList: jspb.Message.toObjectList(msg.getServicesList(),
    proto.io.defang.v1.Service.toObject, includeInstance),
    project: jspb.Message.getFieldWithDefault(msg, 2, ""),
    mode: jspb.Message.getFieldWithDefault(msg, 3, 0),
    compose: msg.getCompose_asB64(),
    delegateDomain: jspb.Message.getFieldWithDefault(msg, 5, ""),
    delegationSetId: jspb.Message.getFieldWithDefault(msg, 6, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DeployRequest}
 */
proto.io.defang.v1.DeployRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DeployRequest;
  return proto.io.defang.v1.DeployRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DeployRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DeployRequest}
 */
proto.io.defang.v1.DeployRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.Service;
      reader.readMessage(value,proto.io.defang.v1.Service.deserializeBinaryFromReader);
      msg.addServices(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 3:
      var value = /** @type {!proto.io.defang.v1.DeploymentMode} */ (reader.readEnum());
      msg.setMode(value);
      break;
    case 4:
      var value = /** @type {!Uint8Array} */ (reader.readBytes());
      msg.setCompose(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setDelegateDomain(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setDelegationSetId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DeployRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DeployRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DeployRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeployRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getServicesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.Service.serializeBinaryToWriter
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getMode();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
  f = message.getCompose_asU8();
  if (f.length > 0) {
    writer.writeBytes(
      4,
      f
    );
  }
  f = message.getDelegateDomain();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getDelegationSetId();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
};


/**
 * repeated Service services = 1;
 * @return {!Array<!proto.io.defang.v1.Service>}
 */
proto.io.defang.v1.DeployRequest.prototype.getServicesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.Service>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.Service, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.Service>} value
 * @return {!proto.io.defang.v1.DeployRequest} returns this
*/
proto.io.defang.v1.DeployRequest.prototype.setServicesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.Service=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Service}
 */
proto.io.defang.v1.DeployRequest.prototype.addServices = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.Service, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.DeployRequest} returns this
 */
proto.io.defang.v1.DeployRequest.prototype.clearServicesList = function() {
  return this.setServicesList([]);
};


/**
 * optional string project = 2;
 * @return {string}
 */
proto.io.defang.v1.DeployRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployRequest} returns this
 */
proto.io.defang.v1.DeployRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional DeploymentMode mode = 3;
 * @return {!proto.io.defang.v1.DeploymentMode}
 */
proto.io.defang.v1.DeployRequest.prototype.getMode = function() {
  return /** @type {!proto.io.defang.v1.DeploymentMode} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.io.defang.v1.DeploymentMode} value
 * @return {!proto.io.defang.v1.DeployRequest} returns this
 */
proto.io.defang.v1.DeployRequest.prototype.setMode = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};


/**
 * optional bytes compose = 4;
 * @return {!(string|Uint8Array)}
 */
proto.io.defang.v1.DeployRequest.prototype.getCompose = function() {
  return /** @type {!(string|Uint8Array)} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * optional bytes compose = 4;
 * This is a type-conversion wrapper around `getCompose()`
 * @return {string}
 */
proto.io.defang.v1.DeployRequest.prototype.getCompose_asB64 = function() {
  return /** @type {string} */ (jspb.Message.bytesAsB64(
      this.getCompose()));
};


/**
 * optional bytes compose = 4;
 * Note that Uint8Array is not supported on all browsers.
 * @see http://caniuse.com/Uint8Array
 * This is a type-conversion wrapper around `getCompose()`
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DeployRequest.prototype.getCompose_asU8 = function() {
  return /** @type {!Uint8Array} */ (jspb.Message.bytesAsU8(
      this.getCompose()));
};


/**
 * @param {!(string|Uint8Array)} value
 * @return {!proto.io.defang.v1.DeployRequest} returns this
 */
proto.io.defang.v1.DeployRequest.prototype.setCompose = function(value) {
  return jspb.Message.setProto3BytesField(this, 4, value);
};


/**
 * optional string delegate_domain = 5;
 * @return {string}
 */
proto.io.defang.v1.DeployRequest.prototype.getDelegateDomain = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployRequest} returns this
 */
proto.io.defang.v1.DeployRequest.prototype.setDelegateDomain = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional string delegation_set_id = 6;
 * @return {string}
 */
proto.io.defang.v1.DeployRequest.prototype.getDelegationSetId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployRequest} returns this
 */
proto.io.defang.v1.DeployRequest.prototype.setDelegationSetId = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.DeployResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DeployResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DeployResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DeployResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeployResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    servicesList: jspb.Message.toObjectList(msg.getServicesList(),
    proto.io.defang.v1.ServiceInfo.toObject, includeInstance),
    etag: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DeployResponse}
 */
proto.io.defang.v1.DeployResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DeployResponse;
  return proto.io.defang.v1.DeployResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DeployResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DeployResponse}
 */
proto.io.defang.v1.DeployResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.ServiceInfo;
      reader.readMessage(value,proto.io.defang.v1.ServiceInfo.deserializeBinaryFromReader);
      msg.addServices(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setEtag(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DeployResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DeployResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DeployResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeployResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getServicesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.ServiceInfo.serializeBinaryToWriter
    );
  }
  f = message.getEtag();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated ServiceInfo services = 1;
 * @return {!Array<!proto.io.defang.v1.ServiceInfo>}
 */
proto.io.defang.v1.DeployResponse.prototype.getServicesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.ServiceInfo>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.ServiceInfo, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.ServiceInfo>} value
 * @return {!proto.io.defang.v1.DeployResponse} returns this
*/
proto.io.defang.v1.DeployResponse.prototype.setServicesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.ServiceInfo=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.ServiceInfo}
 */
proto.io.defang.v1.DeployResponse.prototype.addServices = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.ServiceInfo, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.DeployResponse} returns this
 */
proto.io.defang.v1.DeployResponse.prototype.clearServicesList = function() {
  return this.setServicesList([]);
};


/**
 * optional string etag = 2;
 * @return {string}
 */
proto.io.defang.v1.DeployResponse.prototype.getEtag = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployResponse} returns this
 */
proto.io.defang.v1.DeployResponse.prototype.setEtag = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.DeleteRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DeleteRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DeleteRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DeleteRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeleteRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    namesList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    project: jspb.Message.getFieldWithDefault(msg, 2, ""),
    delegateDomain: jspb.Message.getFieldWithDefault(msg, 3, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DeleteRequest}
 */
proto.io.defang.v1.DeleteRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DeleteRequest;
  return proto.io.defang.v1.DeleteRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DeleteRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DeleteRequest}
 */
proto.io.defang.v1.DeleteRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.addNames(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setDelegateDomain(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DeleteRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DeleteRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DeleteRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeleteRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getNamesList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      1,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getDelegateDomain();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
};


/**
 * repeated string names = 1;
 * @return {!Array<string>}
 */
proto.io.defang.v1.DeleteRequest.prototype.getNamesList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.DeleteRequest} returns this
 */
proto.io.defang.v1.DeleteRequest.prototype.setNamesList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.DeleteRequest} returns this
 */
proto.io.defang.v1.DeleteRequest.prototype.addNames = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.DeleteRequest} returns this
 */
proto.io.defang.v1.DeleteRequest.prototype.clearNamesList = function() {
  return this.setNamesList([]);
};


/**
 * optional string project = 2;
 * @return {string}
 */
proto.io.defang.v1.DeleteRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeleteRequest} returns this
 */
proto.io.defang.v1.DeleteRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string delegate_domain = 3;
 * @return {string}
 */
proto.io.defang.v1.DeleteRequest.prototype.getDelegateDomain = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeleteRequest} returns this
 */
proto.io.defang.v1.DeleteRequest.prototype.setDelegateDomain = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DeleteResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DeleteResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DeleteResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeleteResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    etag: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DeleteResponse}
 */
proto.io.defang.v1.DeleteResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DeleteResponse;
  return proto.io.defang.v1.DeleteResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DeleteResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DeleteResponse}
 */
proto.io.defang.v1.DeleteResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setEtag(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DeleteResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DeleteResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DeleteResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeleteResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEtag();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string etag = 1;
 * @return {string}
 */
proto.io.defang.v1.DeleteResponse.prototype.getEtag = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeleteResponse} returns this
 */
proto.io.defang.v1.DeleteResponse.prototype.setEtag = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GenerateFilesRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GenerateFilesRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GenerateFilesRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    prompt: jspb.Message.getFieldWithDefault(msg, 1, ""),
    language: jspb.Message.getFieldWithDefault(msg, 2, ""),
    agreeTos: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    trainingOptOut: jspb.Message.getBooleanFieldWithDefault(msg, 4, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GenerateFilesRequest}
 */
proto.io.defang.v1.GenerateFilesRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GenerateFilesRequest;
  return proto.io.defang.v1.GenerateFilesRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GenerateFilesRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GenerateFilesRequest}
 */
proto.io.defang.v1.GenerateFilesRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setPrompt(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setLanguage(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setAgreeTos(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setTrainingOptOut(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GenerateFilesRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GenerateFilesRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GenerateFilesRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPrompt();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getLanguage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getAgreeTos();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getTrainingOptOut();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
};


/**
 * optional string prompt = 1;
 * @return {string}
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.getPrompt = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.GenerateFilesRequest} returns this
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.setPrompt = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string language = 2;
 * @return {string}
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.getLanguage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.GenerateFilesRequest} returns this
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.setLanguage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional bool agree_tos = 3;
 * @return {boolean}
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.getAgreeTos = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.GenerateFilesRequest} returns this
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.setAgreeTos = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional bool training_opt_out = 4;
 * @return {boolean}
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.getTrainingOptOut = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.GenerateFilesRequest} returns this
 */
proto.io.defang.v1.GenerateFilesRequest.prototype.setTrainingOptOut = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.File.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.File.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.File} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.File.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    content: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.File}
 */
proto.io.defang.v1.File.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.File;
  return proto.io.defang.v1.File.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.File} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.File}
 */
proto.io.defang.v1.File.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setContent(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.File.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.File.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.File} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.File.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getContent();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.io.defang.v1.File.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.File} returns this
 */
proto.io.defang.v1.File.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string content = 2;
 * @return {string}
 */
proto.io.defang.v1.File.prototype.getContent = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.File} returns this
 */
proto.io.defang.v1.File.prototype.setContent = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.GenerateFilesResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GenerateFilesResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GenerateFilesResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GenerateFilesResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GenerateFilesResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    filesList: jspb.Message.toObjectList(msg.getFilesList(),
    proto.io.defang.v1.File.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GenerateFilesResponse}
 */
proto.io.defang.v1.GenerateFilesResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GenerateFilesResponse;
  return proto.io.defang.v1.GenerateFilesResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GenerateFilesResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GenerateFilesResponse}
 */
proto.io.defang.v1.GenerateFilesResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.File;
      reader.readMessage(value,proto.io.defang.v1.File.deserializeBinaryFromReader);
      msg.addFiles(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GenerateFilesResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GenerateFilesResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GenerateFilesResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GenerateFilesResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getFilesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.File.serializeBinaryToWriter
    );
  }
};


/**
 * repeated File files = 1;
 * @return {!Array<!proto.io.defang.v1.File>}
 */
proto.io.defang.v1.GenerateFilesResponse.prototype.getFilesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.File>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.File, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.File>} value
 * @return {!proto.io.defang.v1.GenerateFilesResponse} returns this
*/
proto.io.defang.v1.GenerateFilesResponse.prototype.setFilesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.File=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.File}
 */
proto.io.defang.v1.GenerateFilesResponse.prototype.addFiles = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.File, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.GenerateFilesResponse} returns this
 */
proto.io.defang.v1.GenerateFilesResponse.prototype.clearFilesList = function() {
  return this.setFilesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.StartGenerateResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.StartGenerateResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.StartGenerateResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.StartGenerateResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    uuid: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.StartGenerateResponse}
 */
proto.io.defang.v1.StartGenerateResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.StartGenerateResponse;
  return proto.io.defang.v1.StartGenerateResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.StartGenerateResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.StartGenerateResponse}
 */
proto.io.defang.v1.StartGenerateResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUuid(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.StartGenerateResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.StartGenerateResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.StartGenerateResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.StartGenerateResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUuid();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string uuid = 1;
 * @return {string}
 */
proto.io.defang.v1.StartGenerateResponse.prototype.getUuid = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.StartGenerateResponse} returns this
 */
proto.io.defang.v1.StartGenerateResponse.prototype.setUuid = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GenerateStatusRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GenerateStatusRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GenerateStatusRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GenerateStatusRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    uuid: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GenerateStatusRequest}
 */
proto.io.defang.v1.GenerateStatusRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GenerateStatusRequest;
  return proto.io.defang.v1.GenerateStatusRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GenerateStatusRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GenerateStatusRequest}
 */
proto.io.defang.v1.GenerateStatusRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUuid(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GenerateStatusRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GenerateStatusRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GenerateStatusRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GenerateStatusRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUuid();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string uuid = 1;
 * @return {string}
 */
proto.io.defang.v1.GenerateStatusRequest.prototype.getUuid = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.GenerateStatusRequest} returns this
 */
proto.io.defang.v1.GenerateStatusRequest.prototype.setUuid = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.UploadURLRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.UploadURLRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.UploadURLRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.UploadURLRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    digest: jspb.Message.getFieldWithDefault(msg, 1, ""),
    project: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.UploadURLRequest}
 */
proto.io.defang.v1.UploadURLRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.UploadURLRequest;
  return proto.io.defang.v1.UploadURLRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.UploadURLRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.UploadURLRequest}
 */
proto.io.defang.v1.UploadURLRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setDigest(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.UploadURLRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.UploadURLRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.UploadURLRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.UploadURLRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDigest();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string digest = 1;
 * @return {string}
 */
proto.io.defang.v1.UploadURLRequest.prototype.getDigest = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.UploadURLRequest} returns this
 */
proto.io.defang.v1.UploadURLRequest.prototype.setDigest = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string project = 2;
 * @return {string}
 */
proto.io.defang.v1.UploadURLRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.UploadURLRequest} returns this
 */
proto.io.defang.v1.UploadURLRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.UploadURLResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.UploadURLResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.UploadURLResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.UploadURLResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    url: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.UploadURLResponse}
 */
proto.io.defang.v1.UploadURLResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.UploadURLResponse;
  return proto.io.defang.v1.UploadURLResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.UploadURLResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.UploadURLResponse}
 */
proto.io.defang.v1.UploadURLResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUrl(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.UploadURLResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.UploadURLResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.UploadURLResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.UploadURLResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUrl();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string url = 1;
 * @return {string}
 */
proto.io.defang.v1.UploadURLResponse.prototype.getUrl = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.UploadURLResponse} returns this
 */
proto.io.defang.v1.UploadURLResponse.prototype.setUrl = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.ServiceInfo.repeatedFields_ = [2,6,7];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.ServiceInfo.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.ServiceInfo.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.ServiceInfo} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ServiceInfo.toObject = function(includeInstance, msg) {
  var f, obj = {
    service: (f = msg.getService()) && proto.io.defang.v1.Service.toObject(includeInstance, f),
    endpointsList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f,
    project: jspb.Message.getFieldWithDefault(msg, 3, ""),
    etag: jspb.Message.getFieldWithDefault(msg, 4, ""),
    status: jspb.Message.getFieldWithDefault(msg, 5, ""),
    natIpsList: (f = jspb.Message.getRepeatedField(msg, 6)) == null ? undefined : f,
    lbIpsList: (f = jspb.Message.getRepeatedField(msg, 7)) == null ? undefined : f,
    privateFqdn: jspb.Message.getFieldWithDefault(msg, 8, ""),
    publicFqdn: jspb.Message.getFieldWithDefault(msg, 9, ""),
    createdAt: (f = msg.getCreatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    updatedAt: (f = msg.getUpdatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    zoneId: jspb.Message.getFieldWithDefault(msg, 12, ""),
    useAcmeCert: jspb.Message.getBooleanFieldWithDefault(msg, 13, false),
    state: jspb.Message.getFieldWithDefault(msg, 15, 0),
    domainname: jspb.Message.getFieldWithDefault(msg, 16, ""),
    lbDnsName: jspb.Message.getFieldWithDefault(msg, 17, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.ServiceInfo}
 */
proto.io.defang.v1.ServiceInfo.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.ServiceInfo;
  return proto.io.defang.v1.ServiceInfo.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.ServiceInfo} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.ServiceInfo}
 */
proto.io.defang.v1.ServiceInfo.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.Service;
      reader.readMessage(value,proto.io.defang.v1.Service.deserializeBinaryFromReader);
      msg.setService(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.addEndpoints(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setEtag(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setStatus(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.addNatIps(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.addLbIps(value);
      break;
    case 8:
      var value = /** @type {string} */ (reader.readString());
      msg.setPrivateFqdn(value);
      break;
    case 9:
      var value = /** @type {string} */ (reader.readString());
      msg.setPublicFqdn(value);
      break;
    case 10:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setCreatedAt(value);
      break;
    case 11:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setUpdatedAt(value);
      break;
    case 12:
      var value = /** @type {string} */ (reader.readString());
      msg.setZoneId(value);
      break;
    case 13:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setUseAcmeCert(value);
      break;
    case 15:
      var value = /** @type {!proto.io.defang.v1.ServiceState} */ (reader.readEnum());
      msg.setState(value);
      break;
    case 16:
      var value = /** @type {string} */ (reader.readString());
      msg.setDomainname(value);
      break;
    case 17:
      var value = /** @type {string} */ (reader.readString());
      msg.setLbDnsName(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.ServiceInfo.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.ServiceInfo.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.ServiceInfo} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ServiceInfo.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getService();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.io.defang.v1.Service.serializeBinaryToWriter
    );
  }
  f = message.getEndpointsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      2,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getEtag();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getStatus();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getNatIpsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      6,
      f
    );
  }
  f = message.getLbIpsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      7,
      f
    );
  }
  f = message.getPrivateFqdn();
  if (f.length > 0) {
    writer.writeString(
      8,
      f
    );
  }
  f = message.getPublicFqdn();
  if (f.length > 0) {
    writer.writeString(
      9,
      f
    );
  }
  f = message.getCreatedAt();
  if (f != null) {
    writer.writeMessage(
      10,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getUpdatedAt();
  if (f != null) {
    writer.writeMessage(
      11,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getZoneId();
  if (f.length > 0) {
    writer.writeString(
      12,
      f
    );
  }
  f = message.getUseAcmeCert();
  if (f) {
    writer.writeBool(
      13,
      f
    );
  }
  f = message.getState();
  if (f !== 0.0) {
    writer.writeEnum(
      15,
      f
    );
  }
  f = message.getDomainname();
  if (f.length > 0) {
    writer.writeString(
      16,
      f
    );
  }
  f = message.getLbDnsName();
  if (f.length > 0) {
    writer.writeString(
      17,
      f
    );
  }
};


/**
 * optional Service service = 1;
 * @return {?proto.io.defang.v1.Service}
 */
proto.io.defang.v1.ServiceInfo.prototype.getService = function() {
  return /** @type{?proto.io.defang.v1.Service} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.Service, 1));
};


/**
 * @param {?proto.io.defang.v1.Service|undefined} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
*/
proto.io.defang.v1.ServiceInfo.prototype.setService = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.clearService = function() {
  return this.setService(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.ServiceInfo.prototype.hasService = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * repeated string endpoints = 2;
 * @return {!Array<string>}
 */
proto.io.defang.v1.ServiceInfo.prototype.getEndpointsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setEndpointsList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.addEndpoints = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.clearEndpointsList = function() {
  return this.setEndpointsList([]);
};


/**
 * optional string project = 3;
 * @return {string}
 */
proto.io.defang.v1.ServiceInfo.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string etag = 4;
 * @return {string}
 */
proto.io.defang.v1.ServiceInfo.prototype.getEtag = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setEtag = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string status = 5;
 * @return {string}
 */
proto.io.defang.v1.ServiceInfo.prototype.getStatus = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setStatus = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * repeated string nat_ips = 6;
 * @return {!Array<string>}
 */
proto.io.defang.v1.ServiceInfo.prototype.getNatIpsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 6));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setNatIpsList = function(value) {
  return jspb.Message.setField(this, 6, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.addNatIps = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 6, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.clearNatIpsList = function() {
  return this.setNatIpsList([]);
};


/**
 * repeated string lb_ips = 7;
 * @return {!Array<string>}
 */
proto.io.defang.v1.ServiceInfo.prototype.getLbIpsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 7));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setLbIpsList = function(value) {
  return jspb.Message.setField(this, 7, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.addLbIps = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 7, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.clearLbIpsList = function() {
  return this.setLbIpsList([]);
};


/**
 * optional string private_fqdn = 8;
 * @return {string}
 */
proto.io.defang.v1.ServiceInfo.prototype.getPrivateFqdn = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 8, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setPrivateFqdn = function(value) {
  return jspb.Message.setProto3StringField(this, 8, value);
};


/**
 * optional string public_fqdn = 9;
 * @return {string}
 */
proto.io.defang.v1.ServiceInfo.prototype.getPublicFqdn = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 9, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setPublicFqdn = function(value) {
  return jspb.Message.setProto3StringField(this, 9, value);
};


/**
 * optional google.protobuf.Timestamp created_at = 10;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.io.defang.v1.ServiceInfo.prototype.getCreatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 10));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
*/
proto.io.defang.v1.ServiceInfo.prototype.setCreatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 10, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.clearCreatedAt = function() {
  return this.setCreatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.ServiceInfo.prototype.hasCreatedAt = function() {
  return jspb.Message.getField(this, 10) != null;
};


/**
 * optional google.protobuf.Timestamp updated_at = 11;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.io.defang.v1.ServiceInfo.prototype.getUpdatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 11));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
*/
proto.io.defang.v1.ServiceInfo.prototype.setUpdatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 11, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.clearUpdatedAt = function() {
  return this.setUpdatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.ServiceInfo.prototype.hasUpdatedAt = function() {
  return jspb.Message.getField(this, 11) != null;
};


/**
 * optional string zone_id = 12;
 * @return {string}
 */
proto.io.defang.v1.ServiceInfo.prototype.getZoneId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 12, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setZoneId = function(value) {
  return jspb.Message.setProto3StringField(this, 12, value);
};


/**
 * optional bool use_acme_cert = 13;
 * @return {boolean}
 */
proto.io.defang.v1.ServiceInfo.prototype.getUseAcmeCert = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 13, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setUseAcmeCert = function(value) {
  return jspb.Message.setProto3BooleanField(this, 13, value);
};


/**
 * optional ServiceState state = 15;
 * @return {!proto.io.defang.v1.ServiceState}
 */
proto.io.defang.v1.ServiceInfo.prototype.getState = function() {
  return /** @type {!proto.io.defang.v1.ServiceState} */ (jspb.Message.getFieldWithDefault(this, 15, 0));
};


/**
 * @param {!proto.io.defang.v1.ServiceState} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setState = function(value) {
  return jspb.Message.setProto3EnumField(this, 15, value);
};


/**
 * optional string domainname = 16;
 * @return {string}
 */
proto.io.defang.v1.ServiceInfo.prototype.getDomainname = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 16, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setDomainname = function(value) {
  return jspb.Message.setProto3StringField(this, 16, value);
};


/**
 * optional string lb_dns_name = 17;
 * @return {string}
 */
proto.io.defang.v1.ServiceInfo.prototype.getLbDnsName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 17, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ServiceInfo} returns this
 */
proto.io.defang.v1.ServiceInfo.prototype.setLbDnsName = function(value) {
  return jspb.Message.setProto3StringField(this, 17, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.Secrets.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Secrets.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Secrets.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Secrets} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Secrets.toObject = function(includeInstance, msg) {
  var f, obj = {
    namesList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    project: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Secrets}
 */
proto.io.defang.v1.Secrets.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Secrets;
  return proto.io.defang.v1.Secrets.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Secrets} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Secrets}
 */
proto.io.defang.v1.Secrets.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.addNames(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Secrets.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Secrets.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Secrets} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Secrets.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getNamesList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      1,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated string names = 1;
 * @return {!Array<string>}
 */
proto.io.defang.v1.Secrets.prototype.getNamesList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.Secrets} returns this
 */
proto.io.defang.v1.Secrets.prototype.setNamesList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Secrets} returns this
 */
proto.io.defang.v1.Secrets.prototype.addNames = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.Secrets} returns this
 */
proto.io.defang.v1.Secrets.prototype.clearNamesList = function() {
  return this.setNamesList([]);
};


/**
 * optional string project = 2;
 * @return {string}
 */
proto.io.defang.v1.Secrets.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Secrets} returns this
 */
proto.io.defang.v1.Secrets.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.SecretValue.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.SecretValue.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.SecretValue} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SecretValue.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    value: jspb.Message.getFieldWithDefault(msg, 2, ""),
    project: jspb.Message.getFieldWithDefault(msg, 3, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.SecretValue}
 */
proto.io.defang.v1.SecretValue.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.SecretValue;
  return proto.io.defang.v1.SecretValue.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.SecretValue} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.SecretValue}
 */
proto.io.defang.v1.SecretValue.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setValue(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.SecretValue.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.SecretValue.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.SecretValue} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SecretValue.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getValue();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.io.defang.v1.SecretValue.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.SecretValue} returns this
 */
proto.io.defang.v1.SecretValue.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string value = 2;
 * @return {string}
 */
proto.io.defang.v1.SecretValue.prototype.getValue = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.SecretValue} returns this
 */
proto.io.defang.v1.SecretValue.prototype.setValue = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string project = 3;
 * @return {string}
 */
proto.io.defang.v1.SecretValue.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.SecretValue} returns this
 */
proto.io.defang.v1.SecretValue.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Config.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Config.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Config} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Config.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    value: jspb.Message.getFieldWithDefault(msg, 2, ""),
    project: jspb.Message.getFieldWithDefault(msg, 3, ""),
    type: jspb.Message.getFieldWithDefault(msg, 4, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Config}
 */
proto.io.defang.v1.Config.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Config;
  return proto.io.defang.v1.Config.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Config} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Config}
 */
proto.io.defang.v1.Config.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setValue(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 4:
      var value = /** @type {!proto.io.defang.v1.ConfigType} */ (reader.readEnum());
      msg.setType(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Config.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Config.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Config} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Config.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getValue();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getType();
  if (f !== 0.0) {
    writer.writeEnum(
      4,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.io.defang.v1.Config.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Config} returns this
 */
proto.io.defang.v1.Config.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string value = 2;
 * @return {string}
 */
proto.io.defang.v1.Config.prototype.getValue = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Config} returns this
 */
proto.io.defang.v1.Config.prototype.setValue = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string project = 3;
 * @return {string}
 */
proto.io.defang.v1.Config.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Config} returns this
 */
proto.io.defang.v1.Config.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional ConfigType type = 4;
 * @return {!proto.io.defang.v1.ConfigType}
 */
proto.io.defang.v1.Config.prototype.getType = function() {
  return /** @type {!proto.io.defang.v1.ConfigType} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {!proto.io.defang.v1.ConfigType} value
 * @return {!proto.io.defang.v1.Config} returns this
 */
proto.io.defang.v1.Config.prototype.setType = function(value) {
  return jspb.Message.setProto3EnumField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.ConfigKey.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.ConfigKey.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.ConfigKey} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ConfigKey.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    project: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.ConfigKey}
 */
proto.io.defang.v1.ConfigKey.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.ConfigKey;
  return proto.io.defang.v1.ConfigKey.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.ConfigKey} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.ConfigKey}
 */
proto.io.defang.v1.ConfigKey.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.ConfigKey.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.ConfigKey.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.ConfigKey} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ConfigKey.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.io.defang.v1.ConfigKey.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ConfigKey} returns this
 */
proto.io.defang.v1.ConfigKey.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string project = 2;
 * @return {string}
 */
proto.io.defang.v1.ConfigKey.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ConfigKey} returns this
 */
proto.io.defang.v1.ConfigKey.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.PutConfigRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.PutConfigRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.PutConfigRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.PutConfigRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    value: jspb.Message.getFieldWithDefault(msg, 2, ""),
    project: jspb.Message.getFieldWithDefault(msg, 3, ""),
    type: jspb.Message.getFieldWithDefault(msg, 4, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.PutConfigRequest}
 */
proto.io.defang.v1.PutConfigRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.PutConfigRequest;
  return proto.io.defang.v1.PutConfigRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.PutConfigRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.PutConfigRequest}
 */
proto.io.defang.v1.PutConfigRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setValue(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 4:
      var value = /** @type {!proto.io.defang.v1.ConfigType} */ (reader.readEnum());
      msg.setType(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.PutConfigRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.PutConfigRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.PutConfigRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.PutConfigRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getValue();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getType();
  if (f !== 0.0) {
    writer.writeEnum(
      4,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.io.defang.v1.PutConfigRequest.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.PutConfigRequest} returns this
 */
proto.io.defang.v1.PutConfigRequest.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string value = 2;
 * @return {string}
 */
proto.io.defang.v1.PutConfigRequest.prototype.getValue = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.PutConfigRequest} returns this
 */
proto.io.defang.v1.PutConfigRequest.prototype.setValue = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string project = 3;
 * @return {string}
 */
proto.io.defang.v1.PutConfigRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.PutConfigRequest} returns this
 */
proto.io.defang.v1.PutConfigRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional ConfigType type = 4;
 * @return {!proto.io.defang.v1.ConfigType}
 */
proto.io.defang.v1.PutConfigRequest.prototype.getType = function() {
  return /** @type {!proto.io.defang.v1.ConfigType} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {!proto.io.defang.v1.ConfigType} value
 * @return {!proto.io.defang.v1.PutConfigRequest} returns this
 */
proto.io.defang.v1.PutConfigRequest.prototype.setType = function(value) {
  return jspb.Message.setProto3EnumField(this, 4, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.GetConfigsRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GetConfigsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GetConfigsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GetConfigsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetConfigsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    configsList: jspb.Message.toObjectList(msg.getConfigsList(),
    proto.io.defang.v1.ConfigKey.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GetConfigsRequest}
 */
proto.io.defang.v1.GetConfigsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GetConfigsRequest;
  return proto.io.defang.v1.GetConfigsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GetConfigsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GetConfigsRequest}
 */
proto.io.defang.v1.GetConfigsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.ConfigKey;
      reader.readMessage(value,proto.io.defang.v1.ConfigKey.deserializeBinaryFromReader);
      msg.addConfigs(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GetConfigsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GetConfigsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GetConfigsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetConfigsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getConfigsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.ConfigKey.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ConfigKey configs = 1;
 * @return {!Array<!proto.io.defang.v1.ConfigKey>}
 */
proto.io.defang.v1.GetConfigsRequest.prototype.getConfigsList = function() {
  return /** @type{!Array<!proto.io.defang.v1.ConfigKey>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.ConfigKey, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.ConfigKey>} value
 * @return {!proto.io.defang.v1.GetConfigsRequest} returns this
*/
proto.io.defang.v1.GetConfigsRequest.prototype.setConfigsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.ConfigKey=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.ConfigKey}
 */
proto.io.defang.v1.GetConfigsRequest.prototype.addConfigs = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.ConfigKey, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.GetConfigsRequest} returns this
 */
proto.io.defang.v1.GetConfigsRequest.prototype.clearConfigsList = function() {
  return this.setConfigsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.GetConfigsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GetConfigsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GetConfigsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GetConfigsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetConfigsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    configsList: jspb.Message.toObjectList(msg.getConfigsList(),
    proto.io.defang.v1.Config.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GetConfigsResponse}
 */
proto.io.defang.v1.GetConfigsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GetConfigsResponse;
  return proto.io.defang.v1.GetConfigsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GetConfigsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GetConfigsResponse}
 */
proto.io.defang.v1.GetConfigsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.Config;
      reader.readMessage(value,proto.io.defang.v1.Config.deserializeBinaryFromReader);
      msg.addConfigs(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GetConfigsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GetConfigsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GetConfigsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetConfigsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getConfigsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.Config.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Config configs = 1;
 * @return {!Array<!proto.io.defang.v1.Config>}
 */
proto.io.defang.v1.GetConfigsResponse.prototype.getConfigsList = function() {
  return /** @type{!Array<!proto.io.defang.v1.Config>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.Config, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.Config>} value
 * @return {!proto.io.defang.v1.GetConfigsResponse} returns this
*/
proto.io.defang.v1.GetConfigsResponse.prototype.setConfigsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.Config=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Config}
 */
proto.io.defang.v1.GetConfigsResponse.prototype.addConfigs = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.Config, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.GetConfigsResponse} returns this
 */
proto.io.defang.v1.GetConfigsResponse.prototype.clearConfigsList = function() {
  return this.setConfigsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.DeleteConfigsRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DeleteConfigsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DeleteConfigsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DeleteConfigsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeleteConfigsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    configsList: jspb.Message.toObjectList(msg.getConfigsList(),
    proto.io.defang.v1.ConfigKey.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DeleteConfigsRequest}
 */
proto.io.defang.v1.DeleteConfigsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DeleteConfigsRequest;
  return proto.io.defang.v1.DeleteConfigsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DeleteConfigsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DeleteConfigsRequest}
 */
proto.io.defang.v1.DeleteConfigsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.ConfigKey;
      reader.readMessage(value,proto.io.defang.v1.ConfigKey.deserializeBinaryFromReader);
      msg.addConfigs(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DeleteConfigsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DeleteConfigsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DeleteConfigsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeleteConfigsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getConfigsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.ConfigKey.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ConfigKey configs = 1;
 * @return {!Array<!proto.io.defang.v1.ConfigKey>}
 */
proto.io.defang.v1.DeleteConfigsRequest.prototype.getConfigsList = function() {
  return /** @type{!Array<!proto.io.defang.v1.ConfigKey>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.ConfigKey, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.ConfigKey>} value
 * @return {!proto.io.defang.v1.DeleteConfigsRequest} returns this
*/
proto.io.defang.v1.DeleteConfigsRequest.prototype.setConfigsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.ConfigKey=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.ConfigKey}
 */
proto.io.defang.v1.DeleteConfigsRequest.prototype.addConfigs = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.ConfigKey, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.DeleteConfigsRequest} returns this
 */
proto.io.defang.v1.DeleteConfigsRequest.prototype.clearConfigsList = function() {
  return this.setConfigsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.ListConfigsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.ListConfigsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.ListConfigsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ListConfigsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    project: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.ListConfigsRequest}
 */
proto.io.defang.v1.ListConfigsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.ListConfigsRequest;
  return proto.io.defang.v1.ListConfigsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.ListConfigsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.ListConfigsRequest}
 */
proto.io.defang.v1.ListConfigsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.ListConfigsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.ListConfigsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.ListConfigsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ListConfigsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string project = 1;
 * @return {string}
 */
proto.io.defang.v1.ListConfigsRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ListConfigsRequest} returns this
 */
proto.io.defang.v1.ListConfigsRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.ListConfigsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.ListConfigsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.ListConfigsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.ListConfigsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ListConfigsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    configsList: jspb.Message.toObjectList(msg.getConfigsList(),
    proto.io.defang.v1.ConfigKey.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.ListConfigsResponse}
 */
proto.io.defang.v1.ListConfigsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.ListConfigsResponse;
  return proto.io.defang.v1.ListConfigsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.ListConfigsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.ListConfigsResponse}
 */
proto.io.defang.v1.ListConfigsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.ConfigKey;
      reader.readMessage(value,proto.io.defang.v1.ConfigKey.deserializeBinaryFromReader);
      msg.addConfigs(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.ListConfigsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.ListConfigsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.ListConfigsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ListConfigsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getConfigsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.ConfigKey.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ConfigKey configs = 1;
 * @return {!Array<!proto.io.defang.v1.ConfigKey>}
 */
proto.io.defang.v1.ListConfigsResponse.prototype.getConfigsList = function() {
  return /** @type{!Array<!proto.io.defang.v1.ConfigKey>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.ConfigKey, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.ConfigKey>} value
 * @return {!proto.io.defang.v1.ListConfigsResponse} returns this
*/
proto.io.defang.v1.ListConfigsResponse.prototype.setConfigsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.ConfigKey=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.ConfigKey}
 */
proto.io.defang.v1.ListConfigsResponse.prototype.addConfigs = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.ConfigKey, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.ListConfigsResponse} returns this
 */
proto.io.defang.v1.ListConfigsResponse.prototype.clearConfigsList = function() {
  return this.setConfigsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Deployment.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Deployment.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Deployment} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Deployment.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    project: jspb.Message.getFieldWithDefault(msg, 2, ""),
    provider: jspb.Message.getFieldWithDefault(msg, 3, ""),
    providerAccountId: jspb.Message.getFieldWithDefault(msg, 4, ""),
    timestamp: (f = msg.getTimestamp()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    action: jspb.Message.getFieldWithDefault(msg, 6, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Deployment}
 */
proto.io.defang.v1.Deployment.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Deployment;
  return proto.io.defang.v1.Deployment.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Deployment} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Deployment}
 */
proto.io.defang.v1.Deployment.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setProvider(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setProviderAccountId(value);
      break;
    case 5:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setTimestamp(value);
      break;
    case 6:
      var value = /** @type {!proto.io.defang.v1.DeploymentAction} */ (reader.readEnum());
      msg.setAction(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Deployment.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Deployment.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Deployment} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Deployment.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getProvider();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getProviderAccountId();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getTimestamp();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getAction();
  if (f !== 0.0) {
    writer.writeEnum(
      6,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.io.defang.v1.Deployment.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Deployment} returns this
 */
proto.io.defang.v1.Deployment.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string project = 2;
 * @return {string}
 */
proto.io.defang.v1.Deployment.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Deployment} returns this
 */
proto.io.defang.v1.Deployment.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string provider = 3;
 * @return {string}
 */
proto.io.defang.v1.Deployment.prototype.getProvider = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Deployment} returns this
 */
proto.io.defang.v1.Deployment.prototype.setProvider = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string provider_account_id = 4;
 * @return {string}
 */
proto.io.defang.v1.Deployment.prototype.getProviderAccountId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Deployment} returns this
 */
proto.io.defang.v1.Deployment.prototype.setProviderAccountId = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional google.protobuf.Timestamp timestamp = 5;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.io.defang.v1.Deployment.prototype.getTimestamp = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 5));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.io.defang.v1.Deployment} returns this
*/
proto.io.defang.v1.Deployment.prototype.setTimestamp = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.Deployment} returns this
 */
proto.io.defang.v1.Deployment.prototype.clearTimestamp = function() {
  return this.setTimestamp(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.Deployment.prototype.hasTimestamp = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional DeploymentAction action = 6;
 * @return {!proto.io.defang.v1.DeploymentAction}
 */
proto.io.defang.v1.Deployment.prototype.getAction = function() {
  return /** @type {!proto.io.defang.v1.DeploymentAction} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {!proto.io.defang.v1.DeploymentAction} value
 * @return {!proto.io.defang.v1.Deployment} returns this
 */
proto.io.defang.v1.Deployment.prototype.setAction = function(value) {
  return jspb.Message.setProto3EnumField(this, 6, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.PutDeploymentRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.PutDeploymentRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.PutDeploymentRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.PutDeploymentRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    deployment: (f = msg.getDeployment()) && proto.io.defang.v1.Deployment.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.PutDeploymentRequest}
 */
proto.io.defang.v1.PutDeploymentRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.PutDeploymentRequest;
  return proto.io.defang.v1.PutDeploymentRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.PutDeploymentRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.PutDeploymentRequest}
 */
proto.io.defang.v1.PutDeploymentRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.Deployment;
      reader.readMessage(value,proto.io.defang.v1.Deployment.deserializeBinaryFromReader);
      msg.setDeployment(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.PutDeploymentRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.PutDeploymentRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.PutDeploymentRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.PutDeploymentRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDeployment();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.io.defang.v1.Deployment.serializeBinaryToWriter
    );
  }
};


/**
 * optional Deployment deployment = 1;
 * @return {?proto.io.defang.v1.Deployment}
 */
proto.io.defang.v1.PutDeploymentRequest.prototype.getDeployment = function() {
  return /** @type{?proto.io.defang.v1.Deployment} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.Deployment, 1));
};


/**
 * @param {?proto.io.defang.v1.Deployment|undefined} value
 * @return {!proto.io.defang.v1.PutDeploymentRequest} returns this
*/
proto.io.defang.v1.PutDeploymentRequest.prototype.setDeployment = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.PutDeploymentRequest} returns this
 */
proto.io.defang.v1.PutDeploymentRequest.prototype.clearDeployment = function() {
  return this.setDeployment(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.PutDeploymentRequest.prototype.hasDeployment = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.ListDeploymentsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.ListDeploymentsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.ListDeploymentsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ListDeploymentsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    project: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.ListDeploymentsRequest}
 */
proto.io.defang.v1.ListDeploymentsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.ListDeploymentsRequest;
  return proto.io.defang.v1.ListDeploymentsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.ListDeploymentsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.ListDeploymentsRequest}
 */
proto.io.defang.v1.ListDeploymentsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.ListDeploymentsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.ListDeploymentsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.ListDeploymentsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ListDeploymentsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string project = 1;
 * @return {string}
 */
proto.io.defang.v1.ListDeploymentsRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ListDeploymentsRequest} returns this
 */
proto.io.defang.v1.ListDeploymentsRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.ListDeploymentsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.ListDeploymentsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.ListDeploymentsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.ListDeploymentsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ListDeploymentsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    deploymentsList: jspb.Message.toObjectList(msg.getDeploymentsList(),
    proto.io.defang.v1.Deployment.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.ListDeploymentsResponse}
 */
proto.io.defang.v1.ListDeploymentsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.ListDeploymentsResponse;
  return proto.io.defang.v1.ListDeploymentsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.ListDeploymentsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.ListDeploymentsResponse}
 */
proto.io.defang.v1.ListDeploymentsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.Deployment;
      reader.readMessage(value,proto.io.defang.v1.Deployment.deserializeBinaryFromReader);
      msg.addDeployments(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.ListDeploymentsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.ListDeploymentsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.ListDeploymentsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ListDeploymentsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDeploymentsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.Deployment.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Deployment deployments = 1;
 * @return {!Array<!proto.io.defang.v1.Deployment>}
 */
proto.io.defang.v1.ListDeploymentsResponse.prototype.getDeploymentsList = function() {
  return /** @type{!Array<!proto.io.defang.v1.Deployment>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.Deployment, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.Deployment>} value
 * @return {!proto.io.defang.v1.ListDeploymentsResponse} returns this
*/
proto.io.defang.v1.ListDeploymentsResponse.prototype.setDeploymentsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.Deployment=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Deployment}
 */
proto.io.defang.v1.ListDeploymentsResponse.prototype.addDeployments = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.Deployment, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.ListDeploymentsResponse} returns this
 */
proto.io.defang.v1.ListDeploymentsResponse.prototype.clearDeploymentsList = function() {
  return this.setDeploymentsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.TokenRequest.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.TokenRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.TokenRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.TokenRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TokenRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    tenant: jspb.Message.getFieldWithDefault(msg, 1, ""),
    authCode: jspb.Message.getFieldWithDefault(msg, 2, ""),
    scopeList: (f = jspb.Message.getRepeatedField(msg, 3)) == null ? undefined : f,
    assertion: jspb.Message.getFieldWithDefault(msg, 4, ""),
    expiresIn: jspb.Message.getFieldWithDefault(msg, 5, 0),
    anonId: jspb.Message.getFieldWithDefault(msg, 6, ""),
    refreshToken: jspb.Message.getFieldWithDefault(msg, 7, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.TokenRequest}
 */
proto.io.defang.v1.TokenRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.TokenRequest;
  return proto.io.defang.v1.TokenRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.TokenRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.TokenRequest}
 */
proto.io.defang.v1.TokenRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTenant(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setAuthCode(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.addScope(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setAssertion(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setExpiresIn(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setAnonId(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setRefreshToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.TokenRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.TokenRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.TokenRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TokenRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTenant();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getAuthCode();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getScopeList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      3,
      f
    );
  }
  f = message.getAssertion();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getExpiresIn();
  if (f !== 0) {
    writer.writeUint32(
      5,
      f
    );
  }
  f = message.getAnonId();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getRefreshToken();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
};


/**
 * optional string tenant = 1;
 * @return {string}
 */
proto.io.defang.v1.TokenRequest.prototype.getTenant = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TokenRequest} returns this
 */
proto.io.defang.v1.TokenRequest.prototype.setTenant = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string auth_code = 2;
 * @return {string}
 */
proto.io.defang.v1.TokenRequest.prototype.getAuthCode = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TokenRequest} returns this
 */
proto.io.defang.v1.TokenRequest.prototype.setAuthCode = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated string scope = 3;
 * @return {!Array<string>}
 */
proto.io.defang.v1.TokenRequest.prototype.getScopeList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 3));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.TokenRequest} returns this
 */
proto.io.defang.v1.TokenRequest.prototype.setScopeList = function(value) {
  return jspb.Message.setField(this, 3, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.TokenRequest} returns this
 */
proto.io.defang.v1.TokenRequest.prototype.addScope = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 3, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.TokenRequest} returns this
 */
proto.io.defang.v1.TokenRequest.prototype.clearScopeList = function() {
  return this.setScopeList([]);
};


/**
 * optional string assertion = 4;
 * @return {string}
 */
proto.io.defang.v1.TokenRequest.prototype.getAssertion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TokenRequest} returns this
 */
proto.io.defang.v1.TokenRequest.prototype.setAssertion = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional uint32 expires_in = 5;
 * @return {number}
 */
proto.io.defang.v1.TokenRequest.prototype.getExpiresIn = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.TokenRequest} returns this
 */
proto.io.defang.v1.TokenRequest.prototype.setExpiresIn = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional string anon_id = 6;
 * @return {string}
 */
proto.io.defang.v1.TokenRequest.prototype.getAnonId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TokenRequest} returns this
 */
proto.io.defang.v1.TokenRequest.prototype.setAnonId = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional string refresh_token = 7;
 * @return {string}
 */
proto.io.defang.v1.TokenRequest.prototype.getRefreshToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TokenRequest} returns this
 */
proto.io.defang.v1.TokenRequest.prototype.setRefreshToken = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.TokenResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.TokenResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.TokenResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TokenResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    accessToken: jspb.Message.getFieldWithDefault(msg, 1, ""),
    refreshToken: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.TokenResponse}
 */
proto.io.defang.v1.TokenResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.TokenResponse;
  return proto.io.defang.v1.TokenResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.TokenResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.TokenResponse}
 */
proto.io.defang.v1.TokenResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setAccessToken(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setRefreshToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.TokenResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.TokenResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.TokenResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TokenResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getAccessToken();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRefreshToken();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string access_token = 1;
 * @return {string}
 */
proto.io.defang.v1.TokenResponse.prototype.getAccessToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TokenResponse} returns this
 */
proto.io.defang.v1.TokenResponse.prototype.setAccessToken = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string refresh_token = 2;
 * @return {string}
 */
proto.io.defang.v1.TokenResponse.prototype.getRefreshToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TokenResponse} returns this
 */
proto.io.defang.v1.TokenResponse.prototype.setRefreshToken = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Status.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Status.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Status} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Status.toObject = function(includeInstance, msg) {
  var f, obj = {
    version: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Status}
 */
proto.io.defang.v1.Status.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Status;
  return proto.io.defang.v1.Status.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Status} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Status}
 */
proto.io.defang.v1.Status.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setVersion(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Status.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Status.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Status} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Status.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getVersion();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string version = 1;
 * @return {string}
 */
proto.io.defang.v1.Status.prototype.getVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Status} returns this
 */
proto.io.defang.v1.Status.prototype.setVersion = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Version.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Version.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Version} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Version.toObject = function(includeInstance, msg) {
  var f, obj = {
    fabric: jspb.Message.getFieldWithDefault(msg, 1, ""),
    cliMin: jspb.Message.getFieldWithDefault(msg, 3, ""),
    pulumiMin: jspb.Message.getFieldWithDefault(msg, 4, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Version}
 */
proto.io.defang.v1.Version.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Version;
  return proto.io.defang.v1.Version.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Version} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Version}
 */
proto.io.defang.v1.Version.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setFabric(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setCliMin(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setPulumiMin(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Version.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Version.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Version} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Version.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getFabric();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getCliMin();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getPulumiMin();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
};


/**
 * optional string fabric = 1;
 * @return {string}
 */
proto.io.defang.v1.Version.prototype.getFabric = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Version} returns this
 */
proto.io.defang.v1.Version.prototype.setFabric = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string cli_min = 3;
 * @return {string}
 */
proto.io.defang.v1.Version.prototype.getCliMin = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Version} returns this
 */
proto.io.defang.v1.Version.prototype.setCliMin = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string pulumi_min = 4;
 * @return {string}
 */
proto.io.defang.v1.Version.prototype.getPulumiMin = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Version} returns this
 */
proto.io.defang.v1.Version.prototype.setPulumiMin = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.TailRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.TailRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.TailRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.TailRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TailRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    servicesList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    since: (f = msg.getSince()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    etag: jspb.Message.getFieldWithDefault(msg, 3, ""),
    project: jspb.Message.getFieldWithDefault(msg, 4, ""),
    logType: jspb.Message.getFieldWithDefault(msg, 5, 0),
    pattern: jspb.Message.getFieldWithDefault(msg, 6, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.TailRequest}
 */
proto.io.defang.v1.TailRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.TailRequest;
  return proto.io.defang.v1.TailRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.TailRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.TailRequest}
 */
proto.io.defang.v1.TailRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.addServices(value);
      break;
    case 2:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setSince(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setEtag(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setLogType(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setPattern(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.TailRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.TailRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.TailRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TailRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getServicesList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      1,
      f
    );
  }
  f = message.getSince();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getEtag();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getLogType();
  if (f !== 0) {
    writer.writeUint32(
      5,
      f
    );
  }
  f = message.getPattern();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
};


/**
 * repeated string services = 1;
 * @return {!Array<string>}
 */
proto.io.defang.v1.TailRequest.prototype.getServicesList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.TailRequest} returns this
 */
proto.io.defang.v1.TailRequest.prototype.setServicesList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.TailRequest} returns this
 */
proto.io.defang.v1.TailRequest.prototype.addServices = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.TailRequest} returns this
 */
proto.io.defang.v1.TailRequest.prototype.clearServicesList = function() {
  return this.setServicesList([]);
};


/**
 * optional google.protobuf.Timestamp since = 2;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.io.defang.v1.TailRequest.prototype.getSince = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 2));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.io.defang.v1.TailRequest} returns this
*/
proto.io.defang.v1.TailRequest.prototype.setSince = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.TailRequest} returns this
 */
proto.io.defang.v1.TailRequest.prototype.clearSince = function() {
  return this.setSince(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.TailRequest.prototype.hasSince = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string etag = 3;
 * @return {string}
 */
proto.io.defang.v1.TailRequest.prototype.getEtag = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TailRequest} returns this
 */
proto.io.defang.v1.TailRequest.prototype.setEtag = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string project = 4;
 * @return {string}
 */
proto.io.defang.v1.TailRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TailRequest} returns this
 */
proto.io.defang.v1.TailRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional uint32 log_type = 5;
 * @return {number}
 */
proto.io.defang.v1.TailRequest.prototype.getLogType = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.TailRequest} returns this
 */
proto.io.defang.v1.TailRequest.prototype.setLogType = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional string pattern = 6;
 * @return {string}
 */
proto.io.defang.v1.TailRequest.prototype.getPattern = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TailRequest} returns this
 */
proto.io.defang.v1.TailRequest.prototype.setPattern = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.LogEntry.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.LogEntry.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.LogEntry} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.LogEntry.toObject = function(includeInstance, msg) {
  var f, obj = {
    message: jspb.Message.getFieldWithDefault(msg, 1, ""),
    timestamp: (f = msg.getTimestamp()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    stderr: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    service: jspb.Message.getFieldWithDefault(msg, 4, ""),
    etag: jspb.Message.getFieldWithDefault(msg, 5, ""),
    host: jspb.Message.getFieldWithDefault(msg, 6, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.LogEntry}
 */
proto.io.defang.v1.LogEntry.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.LogEntry;
  return proto.io.defang.v1.LogEntry.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.LogEntry} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.LogEntry}
 */
proto.io.defang.v1.LogEntry.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    case 2:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setTimestamp(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setStderr(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setService(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setEtag(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setHost(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.LogEntry.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.LogEntry.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.LogEntry} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.LogEntry.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getTimestamp();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getStderr();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getService();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getEtag();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getHost();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
};


/**
 * optional string message = 1;
 * @return {string}
 */
proto.io.defang.v1.LogEntry.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.LogEntry} returns this
 */
proto.io.defang.v1.LogEntry.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional google.protobuf.Timestamp timestamp = 2;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.io.defang.v1.LogEntry.prototype.getTimestamp = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 2));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.io.defang.v1.LogEntry} returns this
*/
proto.io.defang.v1.LogEntry.prototype.setTimestamp = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.LogEntry} returns this
 */
proto.io.defang.v1.LogEntry.prototype.clearTimestamp = function() {
  return this.setTimestamp(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.LogEntry.prototype.hasTimestamp = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional bool stderr = 3;
 * @return {boolean}
 */
proto.io.defang.v1.LogEntry.prototype.getStderr = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.LogEntry} returns this
 */
proto.io.defang.v1.LogEntry.prototype.setStderr = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional string service = 4;
 * @return {string}
 */
proto.io.defang.v1.LogEntry.prototype.getService = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.LogEntry} returns this
 */
proto.io.defang.v1.LogEntry.prototype.setService = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string etag = 5;
 * @return {string}
 */
proto.io.defang.v1.LogEntry.prototype.getEtag = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.LogEntry} returns this
 */
proto.io.defang.v1.LogEntry.prototype.setEtag = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional string host = 6;
 * @return {string}
 */
proto.io.defang.v1.LogEntry.prototype.getHost = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.LogEntry} returns this
 */
proto.io.defang.v1.LogEntry.prototype.setHost = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.TailResponse.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.TailResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.TailResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.TailResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TailResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    entriesList: jspb.Message.toObjectList(msg.getEntriesList(),
    proto.io.defang.v1.LogEntry.toObject, includeInstance),
    service: jspb.Message.getFieldWithDefault(msg, 3, ""),
    etag: jspb.Message.getFieldWithDefault(msg, 4, ""),
    host: jspb.Message.getFieldWithDefault(msg, 5, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.TailResponse}
 */
proto.io.defang.v1.TailResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.TailResponse;
  return proto.io.defang.v1.TailResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.TailResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.TailResponse}
 */
proto.io.defang.v1.TailResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 2:
      var value = new proto.io.defang.v1.LogEntry;
      reader.readMessage(value,proto.io.defang.v1.LogEntry.deserializeBinaryFromReader);
      msg.addEntries(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setService(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setEtag(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setHost(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.TailResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.TailResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.TailResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.TailResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEntriesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      proto.io.defang.v1.LogEntry.serializeBinaryToWriter
    );
  }
  f = message.getService();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getEtag();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getHost();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
};


/**
 * repeated LogEntry entries = 2;
 * @return {!Array<!proto.io.defang.v1.LogEntry>}
 */
proto.io.defang.v1.TailResponse.prototype.getEntriesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.LogEntry>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.LogEntry, 2));
};


/**
 * @param {!Array<!proto.io.defang.v1.LogEntry>} value
 * @return {!proto.io.defang.v1.TailResponse} returns this
*/
proto.io.defang.v1.TailResponse.prototype.setEntriesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.io.defang.v1.LogEntry=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.LogEntry}
 */
proto.io.defang.v1.TailResponse.prototype.addEntries = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.io.defang.v1.LogEntry, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.TailResponse} returns this
 */
proto.io.defang.v1.TailResponse.prototype.clearEntriesList = function() {
  return this.setEntriesList([]);
};


/**
 * optional string service = 3;
 * @return {string}
 */
proto.io.defang.v1.TailResponse.prototype.getService = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TailResponse} returns this
 */
proto.io.defang.v1.TailResponse.prototype.setService = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string etag = 4;
 * @return {string}
 */
proto.io.defang.v1.TailResponse.prototype.getEtag = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TailResponse} returns this
 */
proto.io.defang.v1.TailResponse.prototype.setEtag = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string host = 5;
 * @return {string}
 */
proto.io.defang.v1.TailResponse.prototype.getHost = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.TailResponse} returns this
 */
proto.io.defang.v1.TailResponse.prototype.setHost = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.GetServicesResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GetServicesResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GetServicesResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GetServicesResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetServicesResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    servicesList: jspb.Message.toObjectList(msg.getServicesList(),
    proto.io.defang.v1.ServiceInfo.toObject, includeInstance),
    project: jspb.Message.getFieldWithDefault(msg, 2, ""),
    expiresAt: (f = msg.getExpiresAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GetServicesResponse}
 */
proto.io.defang.v1.GetServicesResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GetServicesResponse;
  return proto.io.defang.v1.GetServicesResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GetServicesResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GetServicesResponse}
 */
proto.io.defang.v1.GetServicesResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.ServiceInfo;
      reader.readMessage(value,proto.io.defang.v1.ServiceInfo.deserializeBinaryFromReader);
      msg.addServices(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 3:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setExpiresAt(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GetServicesResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GetServicesResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GetServicesResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetServicesResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getServicesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.ServiceInfo.serializeBinaryToWriter
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getExpiresAt();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ServiceInfo services = 1;
 * @return {!Array<!proto.io.defang.v1.ServiceInfo>}
 */
proto.io.defang.v1.GetServicesResponse.prototype.getServicesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.ServiceInfo>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.ServiceInfo, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.ServiceInfo>} value
 * @return {!proto.io.defang.v1.GetServicesResponse} returns this
*/
proto.io.defang.v1.GetServicesResponse.prototype.setServicesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.ServiceInfo=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.ServiceInfo}
 */
proto.io.defang.v1.GetServicesResponse.prototype.addServices = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.ServiceInfo, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.GetServicesResponse} returns this
 */
proto.io.defang.v1.GetServicesResponse.prototype.clearServicesList = function() {
  return this.setServicesList([]);
};


/**
 * optional string project = 2;
 * @return {string}
 */
proto.io.defang.v1.GetServicesResponse.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.GetServicesResponse} returns this
 */
proto.io.defang.v1.GetServicesResponse.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional google.protobuf.Timestamp expires_at = 3;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.io.defang.v1.GetServicesResponse.prototype.getExpiresAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 3));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.io.defang.v1.GetServicesResponse} returns this
*/
proto.io.defang.v1.GetServicesResponse.prototype.setExpiresAt = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.GetServicesResponse} returns this
 */
proto.io.defang.v1.GetServicesResponse.prototype.clearExpiresAt = function() {
  return this.setExpiresAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.GetServicesResponse.prototype.hasExpiresAt = function() {
  return jspb.Message.getField(this, 3) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.ProjectUpdate.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.ProjectUpdate.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.ProjectUpdate.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.ProjectUpdate} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ProjectUpdate.toObject = function(includeInstance, msg) {
  var f, obj = {
    servicesList: jspb.Message.toObjectList(msg.getServicesList(),
    proto.io.defang.v1.ServiceInfo.toObject, includeInstance),
    albArn: jspb.Message.getFieldWithDefault(msg, 2, ""),
    project: jspb.Message.getFieldWithDefault(msg, 3, ""),
    compose: msg.getCompose_asB64(),
    cdVersion: jspb.Message.getFieldWithDefault(msg, 5, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.ProjectUpdate}
 */
proto.io.defang.v1.ProjectUpdate.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.ProjectUpdate;
  return proto.io.defang.v1.ProjectUpdate.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.ProjectUpdate} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.ProjectUpdate}
 */
proto.io.defang.v1.ProjectUpdate.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.ServiceInfo;
      reader.readMessage(value,proto.io.defang.v1.ServiceInfo.deserializeBinaryFromReader);
      msg.addServices(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setAlbArn(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    case 4:
      var value = /** @type {!Uint8Array} */ (reader.readBytes());
      msg.setCompose(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setCdVersion(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.ProjectUpdate.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.ProjectUpdate.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.ProjectUpdate} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.ProjectUpdate.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getServicesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.io.defang.v1.ServiceInfo.serializeBinaryToWriter
    );
  }
  f = message.getAlbArn();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getCompose_asU8();
  if (f.length > 0) {
    writer.writeBytes(
      4,
      f
    );
  }
  f = message.getCdVersion();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
};


/**
 * repeated ServiceInfo services = 1;
 * @return {!Array<!proto.io.defang.v1.ServiceInfo>}
 */
proto.io.defang.v1.ProjectUpdate.prototype.getServicesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.ServiceInfo>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.ServiceInfo, 1));
};


/**
 * @param {!Array<!proto.io.defang.v1.ServiceInfo>} value
 * @return {!proto.io.defang.v1.ProjectUpdate} returns this
*/
proto.io.defang.v1.ProjectUpdate.prototype.setServicesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.io.defang.v1.ServiceInfo=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.ServiceInfo}
 */
proto.io.defang.v1.ProjectUpdate.prototype.addServices = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.io.defang.v1.ServiceInfo, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.ProjectUpdate} returns this
 */
proto.io.defang.v1.ProjectUpdate.prototype.clearServicesList = function() {
  return this.setServicesList([]);
};


/**
 * optional string alb_arn = 2;
 * @return {string}
 */
proto.io.defang.v1.ProjectUpdate.prototype.getAlbArn = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ProjectUpdate} returns this
 */
proto.io.defang.v1.ProjectUpdate.prototype.setAlbArn = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string project = 3;
 * @return {string}
 */
proto.io.defang.v1.ProjectUpdate.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ProjectUpdate} returns this
 */
proto.io.defang.v1.ProjectUpdate.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional bytes compose = 4;
 * @return {!(string|Uint8Array)}
 */
proto.io.defang.v1.ProjectUpdate.prototype.getCompose = function() {
  return /** @type {!(string|Uint8Array)} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * optional bytes compose = 4;
 * This is a type-conversion wrapper around `getCompose()`
 * @return {string}
 */
proto.io.defang.v1.ProjectUpdate.prototype.getCompose_asB64 = function() {
  return /** @type {string} */ (jspb.Message.bytesAsB64(
      this.getCompose()));
};


/**
 * optional bytes compose = 4;
 * Note that Uint8Array is not supported on all browsers.
 * @see http://caniuse.com/Uint8Array
 * This is a type-conversion wrapper around `getCompose()`
 * @return {!Uint8Array}
 */
proto.io.defang.v1.ProjectUpdate.prototype.getCompose_asU8 = function() {
  return /** @type {!Uint8Array} */ (jspb.Message.bytesAsU8(
      this.getCompose()));
};


/**
 * @param {!(string|Uint8Array)} value
 * @return {!proto.io.defang.v1.ProjectUpdate} returns this
 */
proto.io.defang.v1.ProjectUpdate.prototype.setCompose = function(value) {
  return jspb.Message.setProto3BytesField(this, 4, value);
};


/**
 * optional string cd_version = 5;
 * @return {string}
 */
proto.io.defang.v1.ProjectUpdate.prototype.getCdVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.ProjectUpdate} returns this
 */
proto.io.defang.v1.ProjectUpdate.prototype.setCdVersion = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GetRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GetRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GetRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    project: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GetRequest}
 */
proto.io.defang.v1.GetRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GetRequest;
  return proto.io.defang.v1.GetRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GetRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GetRequest}
 */
proto.io.defang.v1.GetRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GetRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GetRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GetRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.io.defang.v1.GetRequest.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.GetRequest} returns this
 */
proto.io.defang.v1.GetRequest.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string project = 2;
 * @return {string}
 */
proto.io.defang.v1.GetRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.GetRequest} returns this
 */
proto.io.defang.v1.GetRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.Device.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Device.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Device.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Device} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Device.toObject = function(includeInstance, msg) {
  var f, obj = {
    capabilitiesList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    driver: jspb.Message.getFieldWithDefault(msg, 2, ""),
    count: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Device}
 */
proto.io.defang.v1.Device.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Device;
  return proto.io.defang.v1.Device.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Device} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Device}
 */
proto.io.defang.v1.Device.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.addCapabilities(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDriver(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setCount(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Device.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Device.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Device} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Device.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCapabilitiesList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      1,
      f
    );
  }
  f = message.getDriver();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getCount();
  if (f !== 0) {
    writer.writeUint32(
      3,
      f
    );
  }
};


/**
 * repeated string capabilities = 1;
 * @return {!Array<string>}
 */
proto.io.defang.v1.Device.prototype.getCapabilitiesList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.Device} returns this
 */
proto.io.defang.v1.Device.prototype.setCapabilitiesList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Device} returns this
 */
proto.io.defang.v1.Device.prototype.addCapabilities = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.Device} returns this
 */
proto.io.defang.v1.Device.prototype.clearCapabilitiesList = function() {
  return this.setCapabilitiesList([]);
};


/**
 * optional string driver = 2;
 * @return {string}
 */
proto.io.defang.v1.Device.prototype.getDriver = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Device} returns this
 */
proto.io.defang.v1.Device.prototype.setDriver = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional uint32 count = 3;
 * @return {number}
 */
proto.io.defang.v1.Device.prototype.getCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.Device} returns this
 */
proto.io.defang.v1.Device.prototype.setCount = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.Resource.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Resource.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Resource.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Resource} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Resource.toObject = function(includeInstance, msg) {
  var f, obj = {
    memory: jspb.Message.getFloatingPointFieldWithDefault(msg, 1, 0.0),
    cpus: jspb.Message.getFloatingPointFieldWithDefault(msg, 2, 0.0),
    devicesList: jspb.Message.toObjectList(msg.getDevicesList(),
    proto.io.defang.v1.Device.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Resource}
 */
proto.io.defang.v1.Resource.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Resource;
  return proto.io.defang.v1.Resource.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Resource} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Resource}
 */
proto.io.defang.v1.Resource.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readFloat());
      msg.setMemory(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readFloat());
      msg.setCpus(value);
      break;
    case 3:
      var value = new proto.io.defang.v1.Device;
      reader.readMessage(value,proto.io.defang.v1.Device.deserializeBinaryFromReader);
      msg.addDevices(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Resource.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Resource.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Resource} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Resource.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMemory();
  if (f !== 0.0) {
    writer.writeFloat(
      1,
      f
    );
  }
  f = message.getCpus();
  if (f !== 0.0) {
    writer.writeFloat(
      2,
      f
    );
  }
  f = message.getDevicesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      proto.io.defang.v1.Device.serializeBinaryToWriter
    );
  }
};


/**
 * optional float memory = 1;
 * @return {number}
 */
proto.io.defang.v1.Resource.prototype.getMemory = function() {
  return /** @type {number} */ (jspb.Message.getFloatingPointFieldWithDefault(this, 1, 0.0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.Resource} returns this
 */
proto.io.defang.v1.Resource.prototype.setMemory = function(value) {
  return jspb.Message.setProto3FloatField(this, 1, value);
};


/**
 * optional float cpus = 2;
 * @return {number}
 */
proto.io.defang.v1.Resource.prototype.getCpus = function() {
  return /** @type {number} */ (jspb.Message.getFloatingPointFieldWithDefault(this, 2, 0.0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.Resource} returns this
 */
proto.io.defang.v1.Resource.prototype.setCpus = function(value) {
  return jspb.Message.setProto3FloatField(this, 2, value);
};


/**
 * repeated Device devices = 3;
 * @return {!Array<!proto.io.defang.v1.Device>}
 */
proto.io.defang.v1.Resource.prototype.getDevicesList = function() {
  return /** @type{!Array<!proto.io.defang.v1.Device>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.Device, 3));
};


/**
 * @param {!Array<!proto.io.defang.v1.Device>} value
 * @return {!proto.io.defang.v1.Resource} returns this
*/
proto.io.defang.v1.Resource.prototype.setDevicesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.io.defang.v1.Device=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Device}
 */
proto.io.defang.v1.Resource.prototype.addDevices = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.io.defang.v1.Device, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.Resource} returns this
 */
proto.io.defang.v1.Resource.prototype.clearDevicesList = function() {
  return this.setDevicesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Resources.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Resources.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Resources} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Resources.toObject = function(includeInstance, msg) {
  var f, obj = {
    reservations: (f = msg.getReservations()) && proto.io.defang.v1.Resource.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Resources}
 */
proto.io.defang.v1.Resources.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Resources;
  return proto.io.defang.v1.Resources.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Resources} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Resources}
 */
proto.io.defang.v1.Resources.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.Resource;
      reader.readMessage(value,proto.io.defang.v1.Resource.deserializeBinaryFromReader);
      msg.setReservations(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Resources.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Resources.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Resources} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Resources.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getReservations();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.io.defang.v1.Resource.serializeBinaryToWriter
    );
  }
};


/**
 * optional Resource reservations = 1;
 * @return {?proto.io.defang.v1.Resource}
 */
proto.io.defang.v1.Resources.prototype.getReservations = function() {
  return /** @type{?proto.io.defang.v1.Resource} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.Resource, 1));
};


/**
 * @param {?proto.io.defang.v1.Resource|undefined} value
 * @return {!proto.io.defang.v1.Resources} returns this
*/
proto.io.defang.v1.Resources.prototype.setReservations = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.Resources} returns this
 */
proto.io.defang.v1.Resources.prototype.clearReservations = function() {
  return this.setReservations(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.Resources.prototype.hasReservations = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Deploy.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Deploy.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Deploy} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Deploy.toObject = function(includeInstance, msg) {
  var f, obj = {
    replicas: jspb.Message.getFieldWithDefault(msg, 1, 0),
    resources: (f = msg.getResources()) && proto.io.defang.v1.Resources.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Deploy}
 */
proto.io.defang.v1.Deploy.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Deploy;
  return proto.io.defang.v1.Deploy.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Deploy} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Deploy}
 */
proto.io.defang.v1.Deploy.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setReplicas(value);
      break;
    case 2:
      var value = new proto.io.defang.v1.Resources;
      reader.readMessage(value,proto.io.defang.v1.Resources.deserializeBinaryFromReader);
      msg.setResources(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Deploy.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Deploy.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Deploy} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Deploy.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getReplicas();
  if (f !== 0) {
    writer.writeUint32(
      1,
      f
    );
  }
  f = message.getResources();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.io.defang.v1.Resources.serializeBinaryToWriter
    );
  }
};


/**
 * optional uint32 replicas = 1;
 * @return {number}
 */
proto.io.defang.v1.Deploy.prototype.getReplicas = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.Deploy} returns this
 */
proto.io.defang.v1.Deploy.prototype.setReplicas = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional Resources resources = 2;
 * @return {?proto.io.defang.v1.Resources}
 */
proto.io.defang.v1.Deploy.prototype.getResources = function() {
  return /** @type{?proto.io.defang.v1.Resources} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.Resources, 2));
};


/**
 * @param {?proto.io.defang.v1.Resources|undefined} value
 * @return {!proto.io.defang.v1.Deploy} returns this
*/
proto.io.defang.v1.Deploy.prototype.setResources = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.Deploy} returns this
 */
proto.io.defang.v1.Deploy.prototype.clearResources = function() {
  return this.setResources(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.Deploy.prototype.hasResources = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Port.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Port.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Port} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Port.toObject = function(includeInstance, msg) {
  var f, obj = {
    target: jspb.Message.getFieldWithDefault(msg, 1, 0),
    protocol: jspb.Message.getFieldWithDefault(msg, 2, 0),
    mode: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Port}
 */
proto.io.defang.v1.Port.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Port;
  return proto.io.defang.v1.Port.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Port} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Port}
 */
proto.io.defang.v1.Port.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setTarget(value);
      break;
    case 2:
      var value = /** @type {!proto.io.defang.v1.Protocol} */ (reader.readEnum());
      msg.setProtocol(value);
      break;
    case 3:
      var value = /** @type {!proto.io.defang.v1.Mode} */ (reader.readEnum());
      msg.setMode(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Port.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Port.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Port} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Port.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTarget();
  if (f !== 0) {
    writer.writeUint32(
      1,
      f
    );
  }
  f = message.getProtocol();
  if (f !== 0.0) {
    writer.writeEnum(
      2,
      f
    );
  }
  f = message.getMode();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
};


/**
 * optional uint32 target = 1;
 * @return {number}
 */
proto.io.defang.v1.Port.prototype.getTarget = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.Port} returns this
 */
proto.io.defang.v1.Port.prototype.setTarget = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional Protocol protocol = 2;
 * @return {!proto.io.defang.v1.Protocol}
 */
proto.io.defang.v1.Port.prototype.getProtocol = function() {
  return /** @type {!proto.io.defang.v1.Protocol} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {!proto.io.defang.v1.Protocol} value
 * @return {!proto.io.defang.v1.Port} returns this
 */
proto.io.defang.v1.Port.prototype.setProtocol = function(value) {
  return jspb.Message.setProto3EnumField(this, 2, value);
};


/**
 * optional Mode mode = 3;
 * @return {!proto.io.defang.v1.Mode}
 */
proto.io.defang.v1.Port.prototype.getMode = function() {
  return /** @type {!proto.io.defang.v1.Mode} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.io.defang.v1.Mode} value
 * @return {!proto.io.defang.v1.Port} returns this
 */
proto.io.defang.v1.Port.prototype.setMode = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Secret.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Secret.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Secret} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Secret.toObject = function(includeInstance, msg) {
  var f, obj = {
    source: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Secret}
 */
proto.io.defang.v1.Secret.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Secret;
  return proto.io.defang.v1.Secret.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Secret} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Secret}
 */
proto.io.defang.v1.Secret.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setSource(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Secret.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Secret.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Secret} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Secret.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSource();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string source = 1;
 * @return {string}
 */
proto.io.defang.v1.Secret.prototype.getSource = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Secret} returns this
 */
proto.io.defang.v1.Secret.prototype.setSource = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Build.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Build.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Build} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Build.toObject = function(includeInstance, msg) {
  var f, obj = {
    context: jspb.Message.getFieldWithDefault(msg, 1, ""),
    dockerfile: jspb.Message.getFieldWithDefault(msg, 2, ""),
    argsMap: (f = msg.getArgsMap()) ? f.toObject(includeInstance, undefined) : [],
    shmSize: jspb.Message.getFloatingPointFieldWithDefault(msg, 4, 0.0),
    target: jspb.Message.getFieldWithDefault(msg, 5, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Build}
 */
proto.io.defang.v1.Build.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Build;
  return proto.io.defang.v1.Build.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Build} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Build}
 */
proto.io.defang.v1.Build.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setContext(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDockerfile(value);
      break;
    case 3:
      var value = msg.getArgsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readString, null, "", "");
         });
      break;
    case 4:
      var value = /** @type {number} */ (reader.readFloat());
      msg.setShmSize(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setTarget(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Build.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Build.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Build} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Build.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContext();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDockerfile();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getArgsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(3, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeString);
  }
  f = message.getShmSize();
  if (f !== 0.0) {
    writer.writeFloat(
      4,
      f
    );
  }
  f = message.getTarget();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
};


/**
 * optional string context = 1;
 * @return {string}
 */
proto.io.defang.v1.Build.prototype.getContext = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Build} returns this
 */
proto.io.defang.v1.Build.prototype.setContext = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string dockerfile = 2;
 * @return {string}
 */
proto.io.defang.v1.Build.prototype.getDockerfile = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Build} returns this
 */
proto.io.defang.v1.Build.prototype.setDockerfile = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * map<string, string> args = 3;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,string>}
 */
proto.io.defang.v1.Build.prototype.getArgsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,string>} */ (
      jspb.Message.getMapField(this, 3, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.io.defang.v1.Build} returns this
 */
proto.io.defang.v1.Build.prototype.clearArgsMap = function() {
  this.getArgsMap().clear();
  return this;};


/**
 * optional float shm_size = 4;
 * @return {number}
 */
proto.io.defang.v1.Build.prototype.getShmSize = function() {
  return /** @type {number} */ (jspb.Message.getFloatingPointFieldWithDefault(this, 4, 0.0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.Build} returns this
 */
proto.io.defang.v1.Build.prototype.setShmSize = function(value) {
  return jspb.Message.setProto3FloatField(this, 4, value);
};


/**
 * optional string target = 5;
 * @return {string}
 */
proto.io.defang.v1.Build.prototype.getTarget = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Build} returns this
 */
proto.io.defang.v1.Build.prototype.setTarget = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.HealthCheck.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.HealthCheck.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.HealthCheck.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.HealthCheck} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.HealthCheck.toObject = function(includeInstance, msg) {
  var f, obj = {
    testList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    interval: jspb.Message.getFieldWithDefault(msg, 2, 0),
    timeout: jspb.Message.getFieldWithDefault(msg, 3, 0),
    retries: jspb.Message.getFieldWithDefault(msg, 4, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.HealthCheck}
 */
proto.io.defang.v1.HealthCheck.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.HealthCheck;
  return proto.io.defang.v1.HealthCheck.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.HealthCheck} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.HealthCheck}
 */
proto.io.defang.v1.HealthCheck.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.addTest(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setInterval(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setTimeout(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setRetries(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.HealthCheck.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.HealthCheck.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.HealthCheck} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.HealthCheck.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTestList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      1,
      f
    );
  }
  f = message.getInterval();
  if (f !== 0) {
    writer.writeUint32(
      2,
      f
    );
  }
  f = message.getTimeout();
  if (f !== 0) {
    writer.writeUint32(
      3,
      f
    );
  }
  f = message.getRetries();
  if (f !== 0) {
    writer.writeUint32(
      4,
      f
    );
  }
};


/**
 * repeated string test = 1;
 * @return {!Array<string>}
 */
proto.io.defang.v1.HealthCheck.prototype.getTestList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.HealthCheck} returns this
 */
proto.io.defang.v1.HealthCheck.prototype.setTestList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.HealthCheck} returns this
 */
proto.io.defang.v1.HealthCheck.prototype.addTest = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.HealthCheck} returns this
 */
proto.io.defang.v1.HealthCheck.prototype.clearTestList = function() {
  return this.setTestList([]);
};


/**
 * optional uint32 interval = 2;
 * @return {number}
 */
proto.io.defang.v1.HealthCheck.prototype.getInterval = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.HealthCheck} returns this
 */
proto.io.defang.v1.HealthCheck.prototype.setInterval = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional uint32 timeout = 3;
 * @return {number}
 */
proto.io.defang.v1.HealthCheck.prototype.getTimeout = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.HealthCheck} returns this
 */
proto.io.defang.v1.HealthCheck.prototype.setTimeout = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional uint32 retries = 4;
 * @return {number}
 */
proto.io.defang.v1.HealthCheck.prototype.getRetries = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.io.defang.v1.HealthCheck} returns this
 */
proto.io.defang.v1.HealthCheck.prototype.setRetries = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.Service.repeatedFields_ = [6,9,11,17];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Service.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Service.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Service} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Service.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    image: jspb.Message.getFieldWithDefault(msg, 2, ""),
    platform: jspb.Message.getFieldWithDefault(msg, 3, 0),
    internal: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    deploy: (f = msg.getDeploy()) && proto.io.defang.v1.Deploy.toObject(includeInstance, f),
    portsList: jspb.Message.toObjectList(msg.getPortsList(),
    proto.io.defang.v1.Port.toObject, includeInstance),
    environmentMap: (f = msg.getEnvironmentMap()) ? f.toObject(includeInstance, undefined) : [],
    build: (f = msg.getBuild()) && proto.io.defang.v1.Build.toObject(includeInstance, f),
    secretsList: jspb.Message.toObjectList(msg.getSecretsList(),
    proto.io.defang.v1.Secret.toObject, includeInstance),
    healthcheck: (f = msg.getHealthcheck()) && proto.io.defang.v1.HealthCheck.toObject(includeInstance, f),
    commandList: (f = jspb.Message.getRepeatedField(msg, 11)) == null ? undefined : f,
    domainname: jspb.Message.getFieldWithDefault(msg, 12, ""),
    init: jspb.Message.getBooleanFieldWithDefault(msg, 13, false),
    dnsRole: jspb.Message.getFieldWithDefault(msg, 14, ""),
    staticFiles: (f = msg.getStaticFiles()) && proto.io.defang.v1.StaticFiles.toObject(includeInstance, f),
    networks: jspb.Message.getFieldWithDefault(msg, 16, 0),
    aliasesList: (f = jspb.Message.getRepeatedField(msg, 17)) == null ? undefined : f,
    redis: (f = msg.getRedis()) && proto.io.defang.v1.Redis.toObject(includeInstance, f),
    project: jspb.Message.getFieldWithDefault(msg, 20, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Service}
 */
proto.io.defang.v1.Service.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Service;
  return proto.io.defang.v1.Service.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Service} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Service}
 */
proto.io.defang.v1.Service.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setImage(value);
      break;
    case 3:
      var value = /** @type {!proto.io.defang.v1.Platform} */ (reader.readEnum());
      msg.setPlatform(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setInternal(value);
      break;
    case 5:
      var value = new proto.io.defang.v1.Deploy;
      reader.readMessage(value,proto.io.defang.v1.Deploy.deserializeBinaryFromReader);
      msg.setDeploy(value);
      break;
    case 6:
      var value = new proto.io.defang.v1.Port;
      reader.readMessage(value,proto.io.defang.v1.Port.deserializeBinaryFromReader);
      msg.addPorts(value);
      break;
    case 7:
      var value = msg.getEnvironmentMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readString, null, "", "");
         });
      break;
    case 8:
      var value = new proto.io.defang.v1.Build;
      reader.readMessage(value,proto.io.defang.v1.Build.deserializeBinaryFromReader);
      msg.setBuild(value);
      break;
    case 9:
      var value = new proto.io.defang.v1.Secret;
      reader.readMessage(value,proto.io.defang.v1.Secret.deserializeBinaryFromReader);
      msg.addSecrets(value);
      break;
    case 10:
      var value = new proto.io.defang.v1.HealthCheck;
      reader.readMessage(value,proto.io.defang.v1.HealthCheck.deserializeBinaryFromReader);
      msg.setHealthcheck(value);
      break;
    case 11:
      var value = /** @type {string} */ (reader.readString());
      msg.addCommand(value);
      break;
    case 12:
      var value = /** @type {string} */ (reader.readString());
      msg.setDomainname(value);
      break;
    case 13:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setInit(value);
      break;
    case 14:
      var value = /** @type {string} */ (reader.readString());
      msg.setDnsRole(value);
      break;
    case 15:
      var value = new proto.io.defang.v1.StaticFiles;
      reader.readMessage(value,proto.io.defang.v1.StaticFiles.deserializeBinaryFromReader);
      msg.setStaticFiles(value);
      break;
    case 16:
      var value = /** @type {!proto.io.defang.v1.Network} */ (reader.readEnum());
      msg.setNetworks(value);
      break;
    case 17:
      var value = /** @type {string} */ (reader.readString());
      msg.addAliases(value);
      break;
    case 18:
      var value = new proto.io.defang.v1.Redis;
      reader.readMessage(value,proto.io.defang.v1.Redis.deserializeBinaryFromReader);
      msg.setRedis(value);
      break;
    case 20:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Service.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Service.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Service} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Service.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getImage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getPlatform();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
  f = message.getInternal();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getDeploy();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      proto.io.defang.v1.Deploy.serializeBinaryToWriter
    );
  }
  f = message.getPortsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      6,
      f,
      proto.io.defang.v1.Port.serializeBinaryToWriter
    );
  }
  f = message.getEnvironmentMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(7, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeString);
  }
  f = message.getBuild();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      proto.io.defang.v1.Build.serializeBinaryToWriter
    );
  }
  f = message.getSecretsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      9,
      f,
      proto.io.defang.v1.Secret.serializeBinaryToWriter
    );
  }
  f = message.getHealthcheck();
  if (f != null) {
    writer.writeMessage(
      10,
      f,
      proto.io.defang.v1.HealthCheck.serializeBinaryToWriter
    );
  }
  f = message.getCommandList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      11,
      f
    );
  }
  f = message.getDomainname();
  if (f.length > 0) {
    writer.writeString(
      12,
      f
    );
  }
  f = message.getInit();
  if (f) {
    writer.writeBool(
      13,
      f
    );
  }
  f = message.getDnsRole();
  if (f.length > 0) {
    writer.writeString(
      14,
      f
    );
  }
  f = message.getStaticFiles();
  if (f != null) {
    writer.writeMessage(
      15,
      f,
      proto.io.defang.v1.StaticFiles.serializeBinaryToWriter
    );
  }
  f = message.getNetworks();
  if (f !== 0.0) {
    writer.writeEnum(
      16,
      f
    );
  }
  f = message.getAliasesList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      17,
      f
    );
  }
  f = message.getRedis();
  if (f != null) {
    writer.writeMessage(
      18,
      f,
      proto.io.defang.v1.Redis.serializeBinaryToWriter
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      20,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.io.defang.v1.Service.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string image = 2;
 * @return {string}
 */
proto.io.defang.v1.Service.prototype.getImage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setImage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional Platform platform = 3;
 * @return {!proto.io.defang.v1.Platform}
 */
proto.io.defang.v1.Service.prototype.getPlatform = function() {
  return /** @type {!proto.io.defang.v1.Platform} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.io.defang.v1.Platform} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setPlatform = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};


/**
 * optional bool internal = 4;
 * @return {boolean}
 */
proto.io.defang.v1.Service.prototype.getInternal = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setInternal = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional Deploy deploy = 5;
 * @return {?proto.io.defang.v1.Deploy}
 */
proto.io.defang.v1.Service.prototype.getDeploy = function() {
  return /** @type{?proto.io.defang.v1.Deploy} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.Deploy, 5));
};


/**
 * @param {?proto.io.defang.v1.Deploy|undefined} value
 * @return {!proto.io.defang.v1.Service} returns this
*/
proto.io.defang.v1.Service.prototype.setDeploy = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearDeploy = function() {
  return this.setDeploy(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.Service.prototype.hasDeploy = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * repeated Port ports = 6;
 * @return {!Array<!proto.io.defang.v1.Port>}
 */
proto.io.defang.v1.Service.prototype.getPortsList = function() {
  return /** @type{!Array<!proto.io.defang.v1.Port>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.Port, 6));
};


/**
 * @param {!Array<!proto.io.defang.v1.Port>} value
 * @return {!proto.io.defang.v1.Service} returns this
*/
proto.io.defang.v1.Service.prototype.setPortsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 6, value);
};


/**
 * @param {!proto.io.defang.v1.Port=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Port}
 */
proto.io.defang.v1.Service.prototype.addPorts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 6, opt_value, proto.io.defang.v1.Port, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearPortsList = function() {
  return this.setPortsList([]);
};


/**
 * map<string, string> environment = 7;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,string>}
 */
proto.io.defang.v1.Service.prototype.getEnvironmentMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,string>} */ (
      jspb.Message.getMapField(this, 7, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearEnvironmentMap = function() {
  this.getEnvironmentMap().clear();
  return this;};


/**
 * optional Build build = 8;
 * @return {?proto.io.defang.v1.Build}
 */
proto.io.defang.v1.Service.prototype.getBuild = function() {
  return /** @type{?proto.io.defang.v1.Build} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.Build, 8));
};


/**
 * @param {?proto.io.defang.v1.Build|undefined} value
 * @return {!proto.io.defang.v1.Service} returns this
*/
proto.io.defang.v1.Service.prototype.setBuild = function(value) {
  return jspb.Message.setWrapperField(this, 8, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearBuild = function() {
  return this.setBuild(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.Service.prototype.hasBuild = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * repeated Secret secrets = 9;
 * @return {!Array<!proto.io.defang.v1.Secret>}
 */
proto.io.defang.v1.Service.prototype.getSecretsList = function() {
  return /** @type{!Array<!proto.io.defang.v1.Secret>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.io.defang.v1.Secret, 9));
};


/**
 * @param {!Array<!proto.io.defang.v1.Secret>} value
 * @return {!proto.io.defang.v1.Service} returns this
*/
proto.io.defang.v1.Service.prototype.setSecretsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 9, value);
};


/**
 * @param {!proto.io.defang.v1.Secret=} opt_value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Secret}
 */
proto.io.defang.v1.Service.prototype.addSecrets = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 9, opt_value, proto.io.defang.v1.Secret, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearSecretsList = function() {
  return this.setSecretsList([]);
};


/**
 * optional HealthCheck healthcheck = 10;
 * @return {?proto.io.defang.v1.HealthCheck}
 */
proto.io.defang.v1.Service.prototype.getHealthcheck = function() {
  return /** @type{?proto.io.defang.v1.HealthCheck} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.HealthCheck, 10));
};


/**
 * @param {?proto.io.defang.v1.HealthCheck|undefined} value
 * @return {!proto.io.defang.v1.Service} returns this
*/
proto.io.defang.v1.Service.prototype.setHealthcheck = function(value) {
  return jspb.Message.setWrapperField(this, 10, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearHealthcheck = function() {
  return this.setHealthcheck(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.Service.prototype.hasHealthcheck = function() {
  return jspb.Message.getField(this, 10) != null;
};


/**
 * repeated string command = 11;
 * @return {!Array<string>}
 */
proto.io.defang.v1.Service.prototype.getCommandList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 11));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setCommandList = function(value) {
  return jspb.Message.setField(this, 11, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.addCommand = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 11, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearCommandList = function() {
  return this.setCommandList([]);
};


/**
 * optional string domainname = 12;
 * @return {string}
 */
proto.io.defang.v1.Service.prototype.getDomainname = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 12, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setDomainname = function(value) {
  return jspb.Message.setProto3StringField(this, 12, value);
};


/**
 * optional bool init = 13;
 * @return {boolean}
 */
proto.io.defang.v1.Service.prototype.getInit = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 13, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setInit = function(value) {
  return jspb.Message.setProto3BooleanField(this, 13, value);
};


/**
 * optional string dns_role = 14;
 * @return {string}
 */
proto.io.defang.v1.Service.prototype.getDnsRole = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 14, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setDnsRole = function(value) {
  return jspb.Message.setProto3StringField(this, 14, value);
};


/**
 * optional StaticFiles static_files = 15;
 * @return {?proto.io.defang.v1.StaticFiles}
 */
proto.io.defang.v1.Service.prototype.getStaticFiles = function() {
  return /** @type{?proto.io.defang.v1.StaticFiles} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.StaticFiles, 15));
};


/**
 * @param {?proto.io.defang.v1.StaticFiles|undefined} value
 * @return {!proto.io.defang.v1.Service} returns this
*/
proto.io.defang.v1.Service.prototype.setStaticFiles = function(value) {
  return jspb.Message.setWrapperField(this, 15, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearStaticFiles = function() {
  return this.setStaticFiles(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.Service.prototype.hasStaticFiles = function() {
  return jspb.Message.getField(this, 15) != null;
};


/**
 * optional Network networks = 16;
 * @return {!proto.io.defang.v1.Network}
 */
proto.io.defang.v1.Service.prototype.getNetworks = function() {
  return /** @type {!proto.io.defang.v1.Network} */ (jspb.Message.getFieldWithDefault(this, 16, 0));
};


/**
 * @param {!proto.io.defang.v1.Network} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setNetworks = function(value) {
  return jspb.Message.setProto3EnumField(this, 16, value);
};


/**
 * repeated string aliases = 17;
 * @return {!Array<string>}
 */
proto.io.defang.v1.Service.prototype.getAliasesList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 17));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setAliasesList = function(value) {
  return jspb.Message.setField(this, 17, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.addAliases = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 17, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearAliasesList = function() {
  return this.setAliasesList([]);
};


/**
 * optional Redis redis = 18;
 * @return {?proto.io.defang.v1.Redis}
 */
proto.io.defang.v1.Service.prototype.getRedis = function() {
  return /** @type{?proto.io.defang.v1.Redis} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.Redis, 18));
};


/**
 * @param {?proto.io.defang.v1.Redis|undefined} value
 * @return {!proto.io.defang.v1.Service} returns this
*/
proto.io.defang.v1.Service.prototype.setRedis = function(value) {
  return jspb.Message.setWrapperField(this, 18, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.clearRedis = function() {
  return this.setRedis(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.Service.prototype.hasRedis = function() {
  return jspb.Message.getField(this, 18) != null;
};


/**
 * optional string project = 20;
 * @return {string}
 */
proto.io.defang.v1.Service.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 20, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Service} returns this
 */
proto.io.defang.v1.Service.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 20, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.StaticFiles.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.StaticFiles.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.StaticFiles.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.StaticFiles} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.StaticFiles.toObject = function(includeInstance, msg) {
  var f, obj = {
    folder: jspb.Message.getFieldWithDefault(msg, 1, ""),
    redirectsList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.StaticFiles}
 */
proto.io.defang.v1.StaticFiles.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.StaticFiles;
  return proto.io.defang.v1.StaticFiles.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.StaticFiles} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.StaticFiles}
 */
proto.io.defang.v1.StaticFiles.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setFolder(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.addRedirects(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.StaticFiles.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.StaticFiles.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.StaticFiles} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.StaticFiles.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getFolder();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRedirectsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      2,
      f
    );
  }
};


/**
 * optional string folder = 1;
 * @return {string}
 */
proto.io.defang.v1.StaticFiles.prototype.getFolder = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.StaticFiles} returns this
 */
proto.io.defang.v1.StaticFiles.prototype.setFolder = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated string redirects = 2;
 * @return {!Array<string>}
 */
proto.io.defang.v1.StaticFiles.prototype.getRedirectsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.StaticFiles} returns this
 */
proto.io.defang.v1.StaticFiles.prototype.setRedirectsList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.StaticFiles} returns this
 */
proto.io.defang.v1.StaticFiles.prototype.addRedirects = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.StaticFiles} returns this
 */
proto.io.defang.v1.StaticFiles.prototype.clearRedirectsList = function() {
  return this.setRedirectsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Redis.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Redis.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Redis} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Redis.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Redis}
 */
proto.io.defang.v1.Redis.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Redis;
  return proto.io.defang.v1.Redis.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Redis} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Redis}
 */
proto.io.defang.v1.Redis.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Redis.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Redis.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Redis} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Redis.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DeployEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DeployEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DeployEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeployEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    mode: jspb.Message.getFieldWithDefault(msg, 1, 0),
    type: jspb.Message.getFieldWithDefault(msg, 2, ""),
    source: jspb.Message.getFieldWithDefault(msg, 3, ""),
    id: jspb.Message.getFieldWithDefault(msg, 4, ""),
    datacontenttype: jspb.Message.getFieldWithDefault(msg, 5, ""),
    dataschema: jspb.Message.getFieldWithDefault(msg, 6, ""),
    subject: jspb.Message.getFieldWithDefault(msg, 7, ""),
    time: (f = msg.getTime()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    data: msg.getData_asB64()
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DeployEvent}
 */
proto.io.defang.v1.DeployEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DeployEvent;
  return proto.io.defang.v1.DeployEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DeployEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DeployEvent}
 */
proto.io.defang.v1.DeployEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.io.defang.v1.DeploymentMode} */ (reader.readEnum());
      msg.setMode(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setType(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setSource(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setDatacontenttype(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setDataschema(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setSubject(value);
      break;
    case 8:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setTime(value);
      break;
    case 9:
      var value = /** @type {!Uint8Array} */ (reader.readBytes());
      msg.setData(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DeployEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DeployEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DeployEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DeployEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMode();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
  f = message.getType();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getSource();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getDatacontenttype();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getDataschema();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getSubject();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
  f = message.getTime();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getData_asU8();
  if (f.length > 0) {
    writer.writeBytes(
      9,
      f
    );
  }
};


/**
 * optional DeploymentMode mode = 1;
 * @return {!proto.io.defang.v1.DeploymentMode}
 */
proto.io.defang.v1.DeployEvent.prototype.getMode = function() {
  return /** @type {!proto.io.defang.v1.DeploymentMode} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.io.defang.v1.DeploymentMode} value
 * @return {!proto.io.defang.v1.DeployEvent} returns this
 */
proto.io.defang.v1.DeployEvent.prototype.setMode = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};


/**
 * optional string type = 2;
 * @return {string}
 */
proto.io.defang.v1.DeployEvent.prototype.getType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployEvent} returns this
 */
proto.io.defang.v1.DeployEvent.prototype.setType = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string source = 3;
 * @return {string}
 */
proto.io.defang.v1.DeployEvent.prototype.getSource = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployEvent} returns this
 */
proto.io.defang.v1.DeployEvent.prototype.setSource = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string id = 4;
 * @return {string}
 */
proto.io.defang.v1.DeployEvent.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployEvent} returns this
 */
proto.io.defang.v1.DeployEvent.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string datacontenttype = 5;
 * @return {string}
 */
proto.io.defang.v1.DeployEvent.prototype.getDatacontenttype = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployEvent} returns this
 */
proto.io.defang.v1.DeployEvent.prototype.setDatacontenttype = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional string dataschema = 6;
 * @return {string}
 */
proto.io.defang.v1.DeployEvent.prototype.getDataschema = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployEvent} returns this
 */
proto.io.defang.v1.DeployEvent.prototype.setDataschema = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional string subject = 7;
 * @return {string}
 */
proto.io.defang.v1.DeployEvent.prototype.getSubject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DeployEvent} returns this
 */
proto.io.defang.v1.DeployEvent.prototype.setSubject = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};


/**
 * optional google.protobuf.Timestamp time = 8;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.io.defang.v1.DeployEvent.prototype.getTime = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 8));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.io.defang.v1.DeployEvent} returns this
*/
proto.io.defang.v1.DeployEvent.prototype.setTime = function(value) {
  return jspb.Message.setWrapperField(this, 8, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.DeployEvent} returns this
 */
proto.io.defang.v1.DeployEvent.prototype.clearTime = function() {
  return this.setTime(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.DeployEvent.prototype.hasTime = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional bytes data = 9;
 * @return {!(string|Uint8Array)}
 */
proto.io.defang.v1.DeployEvent.prototype.getData = function() {
  return /** @type {!(string|Uint8Array)} */ (jspb.Message.getFieldWithDefault(this, 9, ""));
};


/**
 * optional bytes data = 9;
 * This is a type-conversion wrapper around `getData()`
 * @return {string}
 */
proto.io.defang.v1.DeployEvent.prototype.getData_asB64 = function() {
  return /** @type {string} */ (jspb.Message.bytesAsB64(
      this.getData()));
};


/**
 * optional bytes data = 9;
 * Note that Uint8Array is not supported on all browsers.
 * @see http://caniuse.com/Uint8Array
 * This is a type-conversion wrapper around `getData()`
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DeployEvent.prototype.getData_asU8 = function() {
  return /** @type {!Uint8Array} */ (jspb.Message.bytesAsU8(
      this.getData()));
};


/**
 * @param {!(string|Uint8Array)} value
 * @return {!proto.io.defang.v1.DeployEvent} returns this
 */
proto.io.defang.v1.DeployEvent.prototype.setData = function(value) {
  return jspb.Message.setProto3BytesField(this, 9, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.Event.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.Event.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.Event} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Event.toObject = function(includeInstance, msg) {
  var f, obj = {
    specversion: jspb.Message.getFieldWithDefault(msg, 1, ""),
    type: jspb.Message.getFieldWithDefault(msg, 2, ""),
    source: jspb.Message.getFieldWithDefault(msg, 3, ""),
    id: jspb.Message.getFieldWithDefault(msg, 4, ""),
    datacontenttype: jspb.Message.getFieldWithDefault(msg, 5, ""),
    dataschema: jspb.Message.getFieldWithDefault(msg, 6, ""),
    subject: jspb.Message.getFieldWithDefault(msg, 7, ""),
    time: (f = msg.getTime()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    data: msg.getData_asB64()
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.Event}
 */
proto.io.defang.v1.Event.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.Event;
  return proto.io.defang.v1.Event.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.Event} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.Event}
 */
proto.io.defang.v1.Event.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setSpecversion(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setType(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setSource(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setDatacontenttype(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setDataschema(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setSubject(value);
      break;
    case 8:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setTime(value);
      break;
    case 9:
      var value = /** @type {!Uint8Array} */ (reader.readBytes());
      msg.setData(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Event.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.Event.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.Event} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.Event.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSpecversion();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getType();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getSource();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getDatacontenttype();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getDataschema();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getSubject();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
  f = message.getTime();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getData_asU8();
  if (f.length > 0) {
    writer.writeBytes(
      9,
      f
    );
  }
};


/**
 * optional string specversion = 1;
 * @return {string}
 */
proto.io.defang.v1.Event.prototype.getSpecversion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Event} returns this
 */
proto.io.defang.v1.Event.prototype.setSpecversion = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string type = 2;
 * @return {string}
 */
proto.io.defang.v1.Event.prototype.getType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Event} returns this
 */
proto.io.defang.v1.Event.prototype.setType = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string source = 3;
 * @return {string}
 */
proto.io.defang.v1.Event.prototype.getSource = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Event} returns this
 */
proto.io.defang.v1.Event.prototype.setSource = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string id = 4;
 * @return {string}
 */
proto.io.defang.v1.Event.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Event} returns this
 */
proto.io.defang.v1.Event.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string datacontenttype = 5;
 * @return {string}
 */
proto.io.defang.v1.Event.prototype.getDatacontenttype = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Event} returns this
 */
proto.io.defang.v1.Event.prototype.setDatacontenttype = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional string dataschema = 6;
 * @return {string}
 */
proto.io.defang.v1.Event.prototype.getDataschema = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Event} returns this
 */
proto.io.defang.v1.Event.prototype.setDataschema = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional string subject = 7;
 * @return {string}
 */
proto.io.defang.v1.Event.prototype.getSubject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.Event} returns this
 */
proto.io.defang.v1.Event.prototype.setSubject = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};


/**
 * optional google.protobuf.Timestamp time = 8;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.io.defang.v1.Event.prototype.getTime = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 8));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.io.defang.v1.Event} returns this
*/
proto.io.defang.v1.Event.prototype.setTime = function(value) {
  return jspb.Message.setWrapperField(this, 8, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.Event} returns this
 */
proto.io.defang.v1.Event.prototype.clearTime = function() {
  return this.setTime(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.Event.prototype.hasTime = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional bytes data = 9;
 * @return {!(string|Uint8Array)}
 */
proto.io.defang.v1.Event.prototype.getData = function() {
  return /** @type {!(string|Uint8Array)} */ (jspb.Message.getFieldWithDefault(this, 9, ""));
};


/**
 * optional bytes data = 9;
 * This is a type-conversion wrapper around `getData()`
 * @return {string}
 */
proto.io.defang.v1.Event.prototype.getData_asB64 = function() {
  return /** @type {string} */ (jspb.Message.bytesAsB64(
      this.getData()));
};


/**
 * optional bytes data = 9;
 * Note that Uint8Array is not supported on all browsers.
 * @see http://caniuse.com/Uint8Array
 * This is a type-conversion wrapper around `getData()`
 * @return {!Uint8Array}
 */
proto.io.defang.v1.Event.prototype.getData_asU8 = function() {
  return /** @type {!Uint8Array} */ (jspb.Message.bytesAsU8(
      this.getData()));
};


/**
 * @param {!(string|Uint8Array)} value
 * @return {!proto.io.defang.v1.Event} returns this
 */
proto.io.defang.v1.Event.prototype.setData = function(value) {
  return jspb.Message.setProto3BytesField(this, 9, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.PublishRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.PublishRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.PublishRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.PublishRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    event: (f = msg.getEvent()) && proto.io.defang.v1.Event.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.PublishRequest}
 */
proto.io.defang.v1.PublishRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.PublishRequest;
  return proto.io.defang.v1.PublishRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.PublishRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.PublishRequest}
 */
proto.io.defang.v1.PublishRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.Event;
      reader.readMessage(value,proto.io.defang.v1.Event.deserializeBinaryFromReader);
      msg.setEvent(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.PublishRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.PublishRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.PublishRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.PublishRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEvent();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.io.defang.v1.Event.serializeBinaryToWriter
    );
  }
};


/**
 * optional Event event = 1;
 * @return {?proto.io.defang.v1.Event}
 */
proto.io.defang.v1.PublishRequest.prototype.getEvent = function() {
  return /** @type{?proto.io.defang.v1.Event} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.Event, 1));
};


/**
 * @param {?proto.io.defang.v1.Event|undefined} value
 * @return {!proto.io.defang.v1.PublishRequest} returns this
*/
proto.io.defang.v1.PublishRequest.prototype.setEvent = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.PublishRequest} returns this
 */
proto.io.defang.v1.PublishRequest.prototype.clearEvent = function() {
  return this.setEvent(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.PublishRequest.prototype.hasEvent = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.SubscribeRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.SubscribeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.SubscribeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.SubscribeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SubscribeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    servicesList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    etag: jspb.Message.getFieldWithDefault(msg, 2, ""),
    project: jspb.Message.getFieldWithDefault(msg, 3, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.SubscribeRequest}
 */
proto.io.defang.v1.SubscribeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.SubscribeRequest;
  return proto.io.defang.v1.SubscribeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.SubscribeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.SubscribeRequest}
 */
proto.io.defang.v1.SubscribeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.addServices(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setEtag(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.SubscribeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.SubscribeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.SubscribeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SubscribeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getServicesList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      1,
      f
    );
  }
  f = message.getEtag();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
};


/**
 * repeated string services = 1;
 * @return {!Array<string>}
 */
proto.io.defang.v1.SubscribeRequest.prototype.getServicesList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.SubscribeRequest} returns this
 */
proto.io.defang.v1.SubscribeRequest.prototype.setServicesList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.SubscribeRequest} returns this
 */
proto.io.defang.v1.SubscribeRequest.prototype.addServices = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.SubscribeRequest} returns this
 */
proto.io.defang.v1.SubscribeRequest.prototype.clearServicesList = function() {
  return this.setServicesList([]);
};


/**
 * optional string etag = 2;
 * @return {string}
 */
proto.io.defang.v1.SubscribeRequest.prototype.getEtag = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.SubscribeRequest} returns this
 */
proto.io.defang.v1.SubscribeRequest.prototype.setEtag = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string project = 3;
 * @return {string}
 */
proto.io.defang.v1.SubscribeRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.SubscribeRequest} returns this
 */
proto.io.defang.v1.SubscribeRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.SubscribeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.SubscribeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.SubscribeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SubscribeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    service: (f = msg.getService()) && proto.io.defang.v1.ServiceInfo.toObject(includeInstance, f),
    name: jspb.Message.getFieldWithDefault(msg, 2, ""),
    status: jspb.Message.getFieldWithDefault(msg, 3, ""),
    state: jspb.Message.getFieldWithDefault(msg, 4, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.SubscribeResponse}
 */
proto.io.defang.v1.SubscribeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.SubscribeResponse;
  return proto.io.defang.v1.SubscribeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.SubscribeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.SubscribeResponse}
 */
proto.io.defang.v1.SubscribeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.io.defang.v1.ServiceInfo;
      reader.readMessage(value,proto.io.defang.v1.ServiceInfo.deserializeBinaryFromReader);
      msg.setService(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setStatus(value);
      break;
    case 4:
      var value = /** @type {!proto.io.defang.v1.ServiceState} */ (reader.readEnum());
      msg.setState(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.SubscribeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.SubscribeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.SubscribeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SubscribeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getService();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.io.defang.v1.ServiceInfo.serializeBinaryToWriter
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getStatus();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getState();
  if (f !== 0.0) {
    writer.writeEnum(
      4,
      f
    );
  }
};


/**
 * optional ServiceInfo service = 1;
 * @return {?proto.io.defang.v1.ServiceInfo}
 */
proto.io.defang.v1.SubscribeResponse.prototype.getService = function() {
  return /** @type{?proto.io.defang.v1.ServiceInfo} */ (
    jspb.Message.getWrapperField(this, proto.io.defang.v1.ServiceInfo, 1));
};


/**
 * @param {?proto.io.defang.v1.ServiceInfo|undefined} value
 * @return {!proto.io.defang.v1.SubscribeResponse} returns this
*/
proto.io.defang.v1.SubscribeResponse.prototype.setService = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.io.defang.v1.SubscribeResponse} returns this
 */
proto.io.defang.v1.SubscribeResponse.prototype.clearService = function() {
  return this.setService(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.io.defang.v1.SubscribeResponse.prototype.hasService = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.io.defang.v1.SubscribeResponse.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.SubscribeResponse} returns this
 */
proto.io.defang.v1.SubscribeResponse.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string status = 3;
 * @return {string}
 */
proto.io.defang.v1.SubscribeResponse.prototype.getStatus = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.SubscribeResponse} returns this
 */
proto.io.defang.v1.SubscribeResponse.prototype.setStatus = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional ServiceState state = 4;
 * @return {!proto.io.defang.v1.ServiceState}
 */
proto.io.defang.v1.SubscribeResponse.prototype.getState = function() {
  return /** @type {!proto.io.defang.v1.ServiceState} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {!proto.io.defang.v1.ServiceState} value
 * @return {!proto.io.defang.v1.SubscribeResponse} returns this
 */
proto.io.defang.v1.SubscribeResponse.prototype.setState = function(value) {
  return jspb.Message.setProto3EnumField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.GetServicesRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.GetServicesRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.GetServicesRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetServicesRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    project: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.GetServicesRequest}
 */
proto.io.defang.v1.GetServicesRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.GetServicesRequest;
  return proto.io.defang.v1.GetServicesRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.GetServicesRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.GetServicesRequest}
 */
proto.io.defang.v1.GetServicesRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProject(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.GetServicesRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.GetServicesRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.GetServicesRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.GetServicesRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProject();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string project = 1;
 * @return {string}
 */
proto.io.defang.v1.GetServicesRequest.prototype.getProject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.GetServicesRequest} returns this
 */
proto.io.defang.v1.GetServicesRequest.prototype.setProject = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DelegateSubdomainZoneRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DelegateSubdomainZoneRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    nameServerRecordsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DelegateSubdomainZoneRequest}
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DelegateSubdomainZoneRequest;
  return proto.io.defang.v1.DelegateSubdomainZoneRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DelegateSubdomainZoneRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DelegateSubdomainZoneRequest}
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.addNameServerRecords(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DelegateSubdomainZoneRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DelegateSubdomainZoneRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getNameServerRecordsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      1,
      f
    );
  }
};


/**
 * repeated string name_server_records = 1;
 * @return {!Array<string>}
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.prototype.getNameServerRecordsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.io.defang.v1.DelegateSubdomainZoneRequest} returns this
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.prototype.setNameServerRecordsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.io.defang.v1.DelegateSubdomainZoneRequest} returns this
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.prototype.addNameServerRecords = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.io.defang.v1.DelegateSubdomainZoneRequest} returns this
 */
proto.io.defang.v1.DelegateSubdomainZoneRequest.prototype.clearNameServerRecordsList = function() {
  return this.setNameServerRecordsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.DelegateSubdomainZoneResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.DelegateSubdomainZoneResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.DelegateSubdomainZoneResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DelegateSubdomainZoneResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    zone: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.DelegateSubdomainZoneResponse}
 */
proto.io.defang.v1.DelegateSubdomainZoneResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.DelegateSubdomainZoneResponse;
  return proto.io.defang.v1.DelegateSubdomainZoneResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.DelegateSubdomainZoneResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.DelegateSubdomainZoneResponse}
 */
proto.io.defang.v1.DelegateSubdomainZoneResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setZone(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.DelegateSubdomainZoneResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.DelegateSubdomainZoneResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.DelegateSubdomainZoneResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.DelegateSubdomainZoneResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getZone();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string zone = 1;
 * @return {string}
 */
proto.io.defang.v1.DelegateSubdomainZoneResponse.prototype.getZone = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.DelegateSubdomainZoneResponse} returns this
 */
proto.io.defang.v1.DelegateSubdomainZoneResponse.prototype.setZone = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.SetOptionsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.SetOptionsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.SetOptionsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SetOptionsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    trainingOptOut: jspb.Message.getBooleanFieldWithDefault(msg, 1, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.SetOptionsRequest}
 */
proto.io.defang.v1.SetOptionsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.SetOptionsRequest;
  return proto.io.defang.v1.SetOptionsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.SetOptionsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.SetOptionsRequest}
 */
proto.io.defang.v1.SetOptionsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setTrainingOptOut(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.SetOptionsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.SetOptionsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.SetOptionsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.SetOptionsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTrainingOptOut();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool training_opt_out = 1;
 * @return {boolean}
 */
proto.io.defang.v1.SetOptionsRequest.prototype.getTrainingOptOut = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.SetOptionsRequest} returns this
 */
proto.io.defang.v1.SetOptionsRequest.prototype.setTrainingOptOut = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.io.defang.v1.WhoAmIResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.io.defang.v1.WhoAmIResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.io.defang.v1.WhoAmIResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.WhoAmIResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    tenant: jspb.Message.getFieldWithDefault(msg, 1, ""),
    account: jspb.Message.getFieldWithDefault(msg, 2, ""),
    region: jspb.Message.getFieldWithDefault(msg, 3, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 4, ""),
    tier: jspb.Message.getFieldWithDefault(msg, 5, 0),
    trainingOptOut: jspb.Message.getBooleanFieldWithDefault(msg, 6, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.io.defang.v1.WhoAmIResponse}
 */
proto.io.defang.v1.WhoAmIResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.io.defang.v1.WhoAmIResponse;
  return proto.io.defang.v1.WhoAmIResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.io.defang.v1.WhoAmIResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.io.defang.v1.WhoAmIResponse}
 */
proto.io.defang.v1.WhoAmIResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTenant(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setAccount(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setRegion(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    case 5:
      var value = /** @type {!proto.io.defang.v1.SubscriptionTier} */ (reader.readEnum());
      msg.setTier(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setTrainingOptOut(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.io.defang.v1.WhoAmIResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.io.defang.v1.WhoAmIResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.io.defang.v1.WhoAmIResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.io.defang.v1.WhoAmIResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTenant();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getAccount();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getRegion();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getTier();
  if (f !== 0.0) {
    writer.writeEnum(
      5,
      f
    );
  }
  f = message.getTrainingOptOut();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
};


/**
 * optional string tenant = 1;
 * @return {string}
 */
proto.io.defang.v1.WhoAmIResponse.prototype.getTenant = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.WhoAmIResponse} returns this
 */
proto.io.defang.v1.WhoAmIResponse.prototype.setTenant = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string account = 2;
 * @return {string}
 */
proto.io.defang.v1.WhoAmIResponse.prototype.getAccount = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.WhoAmIResponse} returns this
 */
proto.io.defang.v1.WhoAmIResponse.prototype.setAccount = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string region = 3;
 * @return {string}
 */
proto.io.defang.v1.WhoAmIResponse.prototype.getRegion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.WhoAmIResponse} returns this
 */
proto.io.defang.v1.WhoAmIResponse.prototype.setRegion = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string user_id = 4;
 * @return {string}
 */
proto.io.defang.v1.WhoAmIResponse.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.io.defang.v1.WhoAmIResponse} returns this
 */
proto.io.defang.v1.WhoAmIResponse.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional SubscriptionTier tier = 5;
 * @return {!proto.io.defang.v1.SubscriptionTier}
 */
proto.io.defang.v1.WhoAmIResponse.prototype.getTier = function() {
  return /** @type {!proto.io.defang.v1.SubscriptionTier} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {!proto.io.defang.v1.SubscriptionTier} value
 * @return {!proto.io.defang.v1.WhoAmIResponse} returns this
 */
proto.io.defang.v1.WhoAmIResponse.prototype.setTier = function(value) {
  return jspb.Message.setProto3EnumField(this, 5, value);
};


/**
 * optional bool training_opt_out = 6;
 * @return {boolean}
 */
proto.io.defang.v1.WhoAmIResponse.prototype.getTrainingOptOut = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.io.defang.v1.WhoAmIResponse} returns this
 */
proto.io.defang.v1.WhoAmIResponse.prototype.setTrainingOptOut = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * @enum {number}
 */
proto.io.defang.v1.Provider = {
  PROVIDER_UNSPECIFIED: 0,
  DEFANG: 1,
  AWS: 2,
  DIGITALOCEAN: 3,
  GCP: 4
};

/**
 * @enum {number}
 */
proto.io.defang.v1.DeploymentMode = {
  UNSPECIFIED_MODE: 0,
  DEVELOPMENT: 1,
  STAGING: 2,
  PRODUCTION: 3
};

/**
 * @enum {number}
 */
proto.io.defang.v1.ServiceState = {
  NOT_SPECIFIED: 0,
  BUILD_QUEUED: 1,
  BUILD_PROVISIONING: 2,
  BUILD_PENDING: 3,
  BUILD_ACTIVATING: 4,
  BUILD_RUNNING: 5,
  BUILD_STOPPING: 6,
  UPDATE_QUEUED: 7,
  DEPLOYMENT_PENDING: 8,
  DEPLOYMENT_COMPLETED: 9,
  DEPLOYMENT_FAILED: 10,
  BUILD_FAILED: 11,
  DEPLOYMENT_SCALED_IN: 12
};

/**
 * @enum {number}
 */
proto.io.defang.v1.ConfigType = {
  CONFIGTYPE_UNSPECIFIED: 0,
  CONFIGTYPE_SENSITIVE: 1
};

/**
 * @enum {number}
 */
proto.io.defang.v1.DeploymentAction = {
  DEPLOYMENT_ACTION_UNSPECIFIED: 0,
  DEPLOYMENT_ACTION_UP: 1,
  DEPLOYMENT_ACTION_DOWN: 2
};

/**
 * @enum {number}
 */
proto.io.defang.v1.Platform = {
  LINUX_AMD64: 0,
  LINUX_ARM64: 1,
  LINUX_ANY: 2
};

/**
 * @enum {number}
 */
proto.io.defang.v1.Protocol = {
  ANY: 0,
  UDP: 1,
  TCP: 2,
  HTTP: 3,
  HTTP2: 4,
  GRPC: 5
};

/**
 * @enum {number}
 */
proto.io.defang.v1.Mode = {
  HOST: 0,
  INGRESS: 1
};

/**
 * @enum {number}
 */
proto.io.defang.v1.Network = {
  UNSPECIFIED: 0,
  PRIVATE: 1,
  PUBLIC: 2
};

/**
 * @enum {number}
 */
proto.io.defang.v1.SubscriptionTier = {
  SUBSCRIPTION_TIER_UNSPECIFIED: 0,
  HOBBY: 1,
  PERSONAL: 2,
  PRO: 3,
  TEAM: 4
};

goog.object.extend(exports, proto.io.defang.v1);
