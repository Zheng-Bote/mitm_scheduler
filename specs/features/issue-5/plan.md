# Feature: Configurable Database Connection Pool (Issue 5)

## Plan
1. **Config Struct**: Update `internal/config/config.go` to add `DbPoolConfig` containing `MaxConns`.
2. **Database Initialization**: Update `internal/db/db.go` to use `cfg.DbPoolConfig.MaxConns` instead of the hardcoded 20. If 0 or missing, default to 20.
3. **SpecDD / SDD**: Update the `scheduler.sdd` or similar files if required to note that pool size is now configurable.
4. **Documentation & Examples**: Update `example_config.json`, `README.md`, and `CHANGELOG.md` to reflect the new `db_pool` or `max_conns` configuration parameter.

## Tasks
- [ ] Task 1: Update `internal/config/config.go`.
- [ ] Task 2: Update `internal/db/db.go`.
- [ ] Task 3: Update `example_config.json`.
- [ ] Task 4: Update `README.md` and `CHANGELOG.md`.
- [ ] Task 5: Check `.sdd` files and update if necessary.
