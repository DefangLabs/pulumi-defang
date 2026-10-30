# Managed models

A Compose file asks for a model by name in its top-level `models:` section, and
services bind to it with `services.<svc>.models`. The providers pick the
concrete cloud model from the deployment's recipe and record the choice, so a
redeployment keeps the model it already has.

This page describes the provider and CD side. User documentation lives in
defang-docs (`docs/concepts/recipe.md`).

## Recipe: `defang-<cloud>:models`

Each cloud reads its own config key: `defang-aws:models`, `defang-gcp:models`,
`defang-azure:models`, `defang-scaleway:models`. The value maps a selector (the
name a Compose file uses) to an ordered list of candidates:

```yaml
config:
  defang-aws:models:
    chat-default:
      - model: us.amazon.nova-2-lite-v1:0
    nova-lite:                       # a family without its revision
      - model: us.amazon.nova-2-lite-v1
        role: chat
  defang-gcp:models:
    chat-default:
      - model: gemini-3.5-flash
        location: global
  defang-azure:models:
    chat-default:                    # first deployable candidate wins
      - {model: gpt-5.1, format: OpenAI, allow-preview: true}
      - {model: gpt-4.1, format: OpenAI, allow-preview: true}
    embedding-pinned:                # one exact revision
      - {model: text-embedding-3-small, format: OpenAI, version: "1", role: embedding}
  defang-scaleway:models:
    chat-default:
      - model: qwen3-235b-a22b-instruct-2507
```

| Field | Clouds | Meaning |
| --- | --- | --- |
| `model` | all | Provider model ID or name. Required. |
| `role` | all | `chat` or `embedding`. Implied for `chat-default`, `chat-large` and `embedding-default`. Required on other keys unless the catalog reports it (AWS, Azure). |
| `location` | GCP | Vertex AI location; defaults to the stack region. |
| `version` | Azure | Exact model revision. A pinned candidate is used or the deployment fails; it never falls through. |
| `format` | Azure | Model format, e.g. `OpenAI`. Required when a name exists in several formats. |
| `sku` | Azure | Deployment SKU. Default: `Standard` if offered, else `GlobalStandard`. |
| `allow-preview` | Azure | Lets a candidate without `version` pick a preview revision. |

Unknown fields, fields on the wrong cloud, empty lists and role conflicts are
errors that name the key and candidate. A built-in alias missing from the
recipe uses the compatibility defaults in each cloud's `models.go`; those match
what earlier CLIs chose.

## Resolution

1. `ai/chat-default`, `ai/chat-large` and `ai/embedding-default` mean the bare
   aliases. Nothing else is rewritten.
2. On AWS an ARN, or an ID with a `:<revision>`, is exact and never looked up
   in the recipe.
3. A recipe key equal to the request supplies the candidates.
4. Otherwise a built-in alias uses its default, and any other name is used as
   a provider model ID.

| Cloud | Without a revision | Catalog |
| --- | --- | --- |
| AWS | `vendor.family-vN` selects the highest active `vendor.family-vN:<n>`; a `us.`/`eu.`/`global.` prefix selects among inference profiles. | `ListFoundationModels`, `ListInferenceProfiles`, only when needed. |
| GCP | IDs are used as Google publishes them. | none |
| Azure | A name selects its highest deployable revision; `10` sorts after `9`. Deprecating and Deprecated revisions are skipped. | the AI Foundry account's model list |
| Scaleway | IDs are used as given; recipe keys map families to IDs. | none |

The next candidate is tried only when the catalog shows a candidate cannot be
deployed. Access, network and quota errors fail the deployment.

## Compose payload

A CLI with the model contract sends standard Compose:

```yaml
models:
  chat:
    model: ai/chat-default
services:
  chat:                       # AWS, GCP, Azure: gateway named after the model
    image: litellm/litellm:v1.92.0
    x-defang-llm: true        # no command: the provider writes it
    ports: [{target: 4000, mode: host, protocol: tcp}]
  app:
    models:
      chat: {model_var: CHAT_MODEL, endpoint_var: CHAT_URL}
    environment:
      CHAT_MODEL: chat-default
      CHAT_URL: http://chat:4000/v1/
```

- AWS and GCP: the gateway runs `--model bedrock/<target>` (or
  `vertex_ai/<target>`) `--alias <name>`. GCP also gets `VERTEXAI_LOCATION`.
  LiteLLM's single-model mode sends every request to that model, so names
  applications were given by earlier deployments keep working.
- Azure: the AI Foundry deployment is named after the request (`chat-default`),
  which is the model name applications send.
- Scaleway: there is no gateway and no model service. The provider writes the
  resolved model ID into each bound `model_var` the service does not already
  set.
- A gateway that still has `--drop_params --model X --alias Y` came from an
  older CLI. It is deployed unchanged and its target is recorded. Any other
  `x-defang-llm` service only gets model access.

## Persistence

Each model gets a `ModelSelection` resource (`defang-<cloud>:index:ModelSelection`,
named after the model). Its inputs are the request and that request's recipe
entry, not the whole recipe.

- Unchanged inputs keep the recorded model, with no catalog call. Refresh never
  re-resolves.
- A changed recipe entry re-resolves a chat model; preview shows the new
  target.
- An embedding model is never replaced silently. A recipe change keeps the
  recorded model and warns. Switching needs a new request that names an exact
  model (a provider ID, or a recipe key with one exact candidate), and the
  stored vectors have to be regenerated.
- On the first deployment with this resource, the CD reads the model each
  service runs from the stack state and seeds the record with it. If an
  embedding model is deployed but its model cannot be read, the deployment
  stops and asks for the model to be pinned.
- Provider aliases such as Vertex `gemini-embedding-001` can still change
  upstream. The record keeps the ID, not the weights.

## CD image and permissions

CD images carry the label `io.defang.model-contract="1"`. The CLI checks it on
the image it is about to run before it sends a payload with model requests.

| Cloud | CD identity needs |
| --- | --- |
| AWS | `bedrock:ListFoundationModels`, `bedrock:ListInferenceProfiles` |
| GCP | nothing new |
| Azure | read access to the AI Foundry account's models and deployments |
| Scaleway | nothing new |
