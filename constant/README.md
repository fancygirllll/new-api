# constant package (`/constant`)

This directory only holds globally reusable **constant definitions** and contains no business logic or dependencies.

## Current files

| File                 | Description                                                                                              |
|----------------------|----------------------------------------------------------------------------------------------------------|
| `azure.go`           | Global constants related to Azure, such as `AzureNoRemoveDotTime` (controls the deadline for removing `.`). |
| `cache_key.go`       | Cache key format strings and token-related field constants, standardizing cache naming rules.              |
| `channel_setting.go` | Channel-level setting keys such as `proxy` and `force_format`.                                             |
| `context_key.go`     | Defines the `ContextKey` type and context key constants used across the project (request time, Token/Channel/User information, etc.). |
| `env.go`             | Global variables for environment configuration, injected from config files or environment variables at startup. |
| `finish_reason.go`   | Set of `finish_reason` string constants returned by OpenAI/GPT requests.                                    |
| `midjourney.go`      | Midjourney error codes, action constants, and model-to-action mappings.                                     |
| `setup.go`           | Indicates whether the project has completed initial setup (the `Setup` boolean).                            |
| `task.go`            | Task platform and action constants plus model-to-action mappings for platforms such as Suno and Midjourney. |
| `user_setting.go`    | User setting key constants and notification types (Email/Webhook, etc.).                                    |

## Usage conventions

1. The `constant` package **may only be imported by other packages**; **importing other custom packages from this package is forbidden**. If truly necessary, only **Go standard library** packages may be imported.
2. No logic related to business processes, database operations, third-party service calls, etc. may be written in this directory.
3. When adding a new type, keep the naming clear and add a description to the **Current files** table in this README so team members can quickly understand its purpose.

> ⚠️ Violating these conventions creates unnecessary coupling between packages and harms maintainability and testability. Please check before committing.
