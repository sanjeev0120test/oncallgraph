# Free stack

Everything in `opsgraph` is free and open-source. There are no accounts, API
keys, secrets, or paid services anywhere in the tool or its CI.

| Concern        | Library / tool                          | License   | Notes                                  |
|----------------|-----------------------------------------|-----------|----------------------------------------|
| CLI framework  | `spf13/cobra`                           | Apache-2.0| commands, flags, help                  |
| Config/YAML    | `gopkg.in/yaml.v3`                       | MIT/Apache| config + fixture parsing               |
| Storage        | `modernc.org/sqlite`                    | BSD-3     | pure Go, **no CGO**                    |
| Git            | `go-git/go-git/v5`                      | Apache-2.0| local repo scanning, no `git` binary   |
| Kubernetes     | YAML snapshot via `yaml.v3`             | -         | pure Go, no `client-go`                |
| AI generation  | `tmc/langchaingo` (llms + ollama)       | MIT       | local Ollama backend                   |
| AI retrieval   | `philippgille/chromem-go`               | MPL-2.0   | pure-Go embeddable vector DB           |
| LLM runtime    | [Ollama](https://ollama.com) (optional) | MIT       | runs models locally; never required    |
| CI             | GitHub Actions                          | -         | free minutes, no secrets               |

## Why these choices

- **No CGO**: `modernc.org/sqlite` means cross-compilation and CI stay trivial
  on Linux/macOS/Windows with `CGO_ENABLED=0`.
- **No `client-go` in v1**: it is heavy and forces Go-version churn, and a live
  cluster can't be validated in secret-free CI. A checked-in snapshot gives the
  same value deterministically.
- **No downloaded tokenizer**: we use a character budget for prompt sizing so
  the AI path needs zero network beyond the local Ollama daemon.
- **Offline by default**: the entire core validates from checked-in fixtures.

## AI is always optional

`--ai` tries a local Ollama model; if it is not installed or reachable, it falls
back to a deterministic, offline, extractive summary. The tool is fully useful
with no AI at all.
