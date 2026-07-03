# Cognito App Client Provisioning for `septiembre login`

`septiembre login` authenticates users via Amazon Cognito Hosted UI using the
OAuth 2.0 Authorization Code flow with PKCE (S256). This requires a **public**
Cognito app client (no client secret) registered against the existing user
pool before the command can be used end-to-end.

This is a one-time, out-of-repo provisioning step run by an operator with AWS
credentials for the Septiembre Cognito user pool. It is NOT run by the CLI or
any CI pipeline.

## Prerequisites

- AWS CLI configured with credentials for the account that owns the user pool.
- User pool ID: `us-east-2_gT2524tBC`.

## Create the app client

```bash
aws cognito-idp create-user-pool-client \
  --user-pool-id us-east-2_gT2524tBC \
  --client-name septiembre-cli \
  --no-generate-secret \
  --allowed-o-auth-flows code \
  --allowed-o-auth-flows-user-pool-client \
  --allowed-o-auth-scopes openid email profile \
  --supported-identity-providers COGNITO \
  --callback-urls \
    "http://127.0.0.1:8976/callback" \
    "http://127.0.0.1:8977/callback" \
    "http://127.0.0.1:8978/callback"
```

Key flags, and why they matter:

- `--no-generate-secret`: the CLI is a public client. There is no client
  secret to protect, so none is generated or stored.
- `--allowed-o-auth-flows code`: Authorization Code grant, exchanged with a
  PKCE `code_verifier` instead of a client secret.
- `--allowed-o-auth-scopes openid email profile`: the minimum scopes needed
  to mint an `id_token` carrying an email/profile claim for the cloud-api
  token-minting call.
- `--callback-urls`: MUST list all three loopback redirect URIs exactly.
  Cognito performs an **exact match** on `redirect_uri` — no wildcards, no
  ephemeral ports. The CLI's loopback server always binds one of
  `127.0.0.1:8976`, `:8977`, or `:8978` (first free wins), so all three must
  be pre-registered.

The command prints the new `ClientId` in its JSON response. Note it down —
it becomes the `-ldflags` build-time default described below.

## Wiring the client ID into the CLI build

The CLI resolves the Cognito client ID at runtime with the same precedence
pattern used for `APIBaseURL`:

1. `SEPTIEMBRE_COGNITO_CLIENT_ID` environment variable
2. `cognito_client_id` key in the config file
3. `credentials.DefaultCognitoClientID` — a build-time default injected via
   `-ldflags` (see `.goreleaser.yaml`), analogous to `internal/version.Version`.

Release builds produced by `goreleaser` inject the client ID automatically;
no manual step is required per release once the `-ldflags` entry below is in
place. Local development builds fall back to the empty default and must
supply `SEPTIEMBRE_COGNITO_CLIENT_ID` (or the config key) to exercise
`septiembre login` against a real Cognito pool.

## Rotation / revocation

To rotate the client ID (e.g. after a suspected leak — note the client ID is
not a secret, but rotation may still be desired for hygiene), create a new
app client with the same settings above, update the `-ldflags` value in
`.goreleaser.yaml`, and cut a new release. The old client ID can then be
deleted from the user pool.
