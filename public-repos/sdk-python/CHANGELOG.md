# Changelog

## 1.12.0 - UNRELEASED

- Capture HTTP response headers in UserCloudsSDKError
- Add "aws-eu-west-1" as available user data region
- Add table as attribute of Column model
- Add region as optional argument for ExecuteAccessor and ExecuteMutator to restrict those calls to only act in the specified data region
- Add access_primary_db_only as optional argument for ExecuteAccessor. By default ExecuteAccessor will use local replicas to speed up processing, but passing in true for access_primary_db_only forces the server to read from the primary db of each user for the call
- Modify userstore_sample.py to verify that the pagination responses are always returned with all fields, instead of omitting empty ones

## 1.11.0 - 21-1-2024

- Move references from single token access policy in accessor to individual token access policies per column via ColumnOutputConfig

## 1.10.0 - 12-09-2024

- Add version to Transformer model, add client methods GetTransformer and UpdateTransformer, and fix a bug in GetAccessPolicy and GetAccessPolicyTemplate
- Add optional search_indexed field to Column model and optional use_search_index field to Accessor model
- New automated release process

## 1.9.2 - 06-08-2024

- Fix missing parameter in \_post_async for AsyncClient

## 1.9.1 - 05-08-2024

- Fix bad token expiration check

## 1.9.0 - 30-07-2024

- Remove optional "email" argument from ListUsers, since the handler ignores it.
- Add optional "organization_id" argument to ListUsers, since the handler supports it.
- Add optional "sort_key", "sort_order", and "filter" pagination arguments to all non-accessor paginated methods.
- Add optional "sort_key" and "sort_order" arguments to ExecuteAccessor.
- Update userstore sample to exercise multi-key accessor pagination.

## 1.8.0 - 19-07-2024

- Breaking change: Add "ending_before" argument to all paginated methods, add pagination to ExecuteAccessor, change "starting_after" and "ending_before" to be str instead of uuid
- Add the ability to cache the access token globally (in process) and share it across clients instances. Pass the optional `use_global_cache_for_token=true` to the client in order to enable.
- Update tokenizer_sample.py to properly create test Transformer

## 1.7.0 - 09-05-2024

- Update userstore sample to exercise partial update columns
- Add methods for creating, retrieving, updating, and deleting ColumnDataTypes
- Add data_type field to Column that refers to a ColumnDataType
- Add input_data_type and output_data_type fields to Transformer that refer to ColumnDataTypes
- Add ColumnDataType resource IDs for native ColumnDataTypes
- Update userstore_sample.py to interact with ColumnDataTypes

## 1.6.1 - Not published

- Add SDK method for data import via ExecuteMutator

## 1.6.0 - 25-03-2025

- Add methods to retrieve and set external OIDC issuers for a tenant

## 1.5.2 - 20-02-2024

- Update SaveUserstoreSDKAsync to use DownloadUserstoreSDKAsync, and allow passing include_example argument to those APIs.
- Add `__repr__` and `__str__` methods to some models to improve DX.

## 1.5.1 - 15-02-2024

- Fix minor issue with userstore_sample.py

## 1.5.0 - 15-02-2024

- Add GetConsentedPurposesForUser method to the Python SDK Client.
- Add `__repr__` and `__str__` methods to some models to improve DX.
- Add `to_dict()` method to the Address class to make it easier to serialize it into user row when creating or modifying a user.
- Add Region enum and move a lot of other constants into enums (AuthnType, ColumnIndexType, DataLifeCycleState, DataType, PolicyType, TransformType).
- Add `data_life_cycle_state` to Accessor model.
- Breaking change: ColumnField parameter "optional" has been changed to "required", with fields not required by default
- Add "input_type_constraints" and "output_type_constraints" parameters of type ColumnConstraints to Transformer
- Add `SaveUserstoreSDK` that saves the generated userstore python SDK to a file.

## 1.4.0 - 02-02-2024

- Breaking change: ColumnInputConfig now has data member "normalizer", previously referred to as "validator"
- Add "User-Agent" & "X-Usercloudssdk-Version" headers to all outgoing requests.
- Add an optional `session_name` kwarg to the `Client` constructor to allow extra information to be incremented into the User-Agent header.
- Add "region" parameter to CreateUser and CreateUserWithMutator to allow specifying in which region the user data should reside
- Rename "compositeunion" and "compositeintersection" types of access policies to "composite_or" and "composite_and"
- Breaking change: Added AsyncClient class and UCHttpAsyncClient interface to support asynchronous requests. As part of this, Error class was consolidated into UserCloudsSDKError, which may break any customers referencing it.
- Add support for column constraints, which enables custom composite columns as well as checks for immutability, uniqueness, and unique IDs within array columns
- Deprecate TENANT_URL, CLIENT_ID and CLIENT_SECRET in favor of USERCLOUDS_TENANT_URL, USERCLOUDS_CLIENT_ID and USERCLOUDS_CLIENT_SECRET environment variables. The deprecated environment variables will be removed in a future release (they still work in this release).

## 1.3.0 - 11-12-2023

- Introduce UCHttpClient to assist in custom HTTP clients by defining the interface in which UserClouds makes requests to [httpx](https://www.python-httpx.org/)
- Changing POST body format for authorization Create functions, and adding if_not_exists option to CreateOrganization. These are non-breaking changes, but older clients using the previous format will be deprecated eventually.

## 1.2.0 - 21-11-2023

- Cleanup httpx client usage.
  This is a breaking change to the HTTP client interface since we stopped passing the deprecated `data` argument to the httpx client methods (PUT & POST) and pass the `content` argument instead.
- Add more type annotations to models and add a couple of `__str__` methods to models.
- Add `include_example` argument to `DownloadUserstoreSDK` method.

## 1.1.1 - 08-11-2023

- Bring back passing kwargs to the HTTP client methods.

## 1.1.0 -- 08-11-2023

- Add CreateUserWithMutator method to Python SDK Client and example for how to use in userstore_sample.py

## 1.0.15 -- 30-10-2023

- Fix future compatibility (make sure the SDK doesn't break if server sends new fields).
- Fix handling HTTP 404 responses to some HTTP DELETE API calls.
- Add support for output_type and reuse_existing_token fields in transformers.
- Lazily request access token when needed instead of on client creation.
- Improve HTTP error handling in SDK

## 1.0.14 -- 12-10-2023

- Added a changelog
- Switched to using [httpx](https://www.python-httpx.org/) for HTTP requests instead of [requests](https://requests.readthedocs.io/en/master/).
- Allow overriding the default HTTP client with a custom one.
- Add SDK methods for managing retention durations for soft-deleted data.
- Various other cleanup to the code.
- Method in the new Python SDK Client for downloading the custom userstore sdk for your userstore (DownloadUserstoreSDK).
- Userstore SDK now includes methods like UpdateUserForPurposes which allows you to pass the purposes in as an array of enum constants.
