# Container Startup Latency

## Measured phases

Timing is logged at `slog.Debug` (per-phase) and `slog.Info` (totals) under the
`startup` message key. Run with `-v` to see phase breakdown.

| Phase | Typical range | Notes |
|---|---|---|
| `image_check_ok` | 0–800 ms | `docker manifest inspect` network call; cached for 5 min after first check |
| `image_build` | 10–60 s | Only when base image or SSH keys change |
| `docker_run` | 100–400 ms | `docker run -d` returns after container is created |
| `inspect_ports` | 50–200 ms | Two `docker inspect` calls for SSH port + creation time |
| `branch_alloc` | 200–800 ms | git fetch + branch create; runs concurrently with docker_run |
| `ssh_wait` | 500 ms–3 s | Polling loop with 10 ms → 100 ms backoff; container must boot sshd |
| `git_init` | 50–200 ms | One SSH round-trip |
| `git_push_branch` | 200 ms–5 s | Scales with repo size (no `--tags`) |
| `sync_default_branch` | 200 ms–4 s | Runs in parallel with `git_push_branch` |
| `git_switch_base` | 50–150 ms | One SSH round-trip (batched with push goroutine) |
| `push_submodules` | 0–10 s | Only when submodules exist |
| `set_origin` | 50–200 ms | `resolveDefaults` + `git remote add` |
| **`run_container_total`** | **1–12 s** | docker_run through set_origin |
| **`container_start_total`** | **1.5–13 s** | image_check through run_container_total |

## Implemented optimizations

### 1. Overlap branch allocation with container SSH boot

**Impact: 300–800 ms**

`caic` now reserves the branch number (just `nextID++`, under `branchMu`) then
runs `Container.StartPhaseA` and `git fetch + branch create` concurrently via
`errgroup`. `StartPhaseA` does image check/build + `docker run` + SSH config.
`StartPhaseB` (SSH wait + git push) starts only after both goroutines complete,
by which time the branch already exists locally.

`Container.Start` in `md` is now a wrapper for `StartPhaseA` + `StartPhaseB`
so existing callers are unaffected.

### 2. Parallelize the two git pushes

**Impact: 200 ms–4 s**

`git_push_branch` and `sync_default_branch` (`resolveDefaults` +
`SyncDefaultBranch`) now run in an `errgroup` inside `runContainerPhaseB`.
They push to different refs (`refs/heads/<branch>` vs `refs/heads/main`) so
there is no conflict. After `eg.Wait()`, `c.DefaultRemote` is populated for
the `set_origin` step.

### 3. SSH poll backoff

**Impact: 50–200 ms**

`runContainerPhaseB` now starts polling at 10 ms and doubles on each miss up
to a 100 ms cap, instead of a flat 100 ms interval. Containers that respond
in 500–600 ms now save 2–3 missed intervals.

### 4. Cache the remote image digest check

**Impact: 200–800 ms per start**

`imageBuildNeeded` calls `cachedRemoteConfigDigest` instead of
`getRemoteConfigDigest` directly. The cache is an in-process `sync.Mutex`-
guarded map keyed on `(rt, image, arch)` with a 5-minute TTL. The first start
pays the registry round-trip; subsequent starts within the TTL skip it.

### 5. Drop `--tags` from git push

**Impact: 100 ms–2 s for tag-heavy repos**

`--tags` was removed from the task branch push and extra-repo pushes in
`runContainerPhaseB`. Tags are not used by agents or by `md diff`/`md pull`.

## Remaining opportunities

### 6. Merge `git init` SSH call into the switch command

**Impact: 50–100 ms**

Currently two sequential SSH connections:
1. `git init -q ~/src/<repo>`
2. `git switch -q <branch> && git branch -f base ...`

These can be merged by passing a custom `--receive-pack` to `git push` that
initializes the repo as part of receiving the push:

```bash
git push --receive-pack="git init -q ~/src/<repo> && git-receive-pack" <remote> <branch>
```

This saves one TCP handshake + SSH key negotiation round-trip.
