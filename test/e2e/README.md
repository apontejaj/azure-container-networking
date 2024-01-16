# ACN E2E

## Objectives
- Steps are reusable
- Steps parameters are saved to the context of the job
- Once written to the job context, the values are immutable
- Cluster resources used in code should be able to be generated to yaml for easy manual repro
- Avoid shell/ps calls wherever possible and use go libraries for typed parameters (avoid capturing error codes/stderr/stdout)

---
## Starter Example:

./hubble/index_test.go
