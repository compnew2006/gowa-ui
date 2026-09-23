# Capturing Go runtime profiles

The application includes an optional pprof listener for diagnosing CPU,
memory, goroutine, and contention problems. It is disabled by default and runs
on a separate HTTP server. Enable it only for a short diagnostic window.

## Enable profiling

Set the following in `config.toml`:

```toml
[profiling]
enabled = true
address = "127.0.0.1:6060"
```

The address must contain a literal loopback IP (`127.0.0.1` or `::1`) and a
port. Hostnames such as `localhost`, wildcard addresses, and non-loopback IPs
are rejected. The pprof routes are not added to the application HTTP server.

The server enables mutex sampling at fraction 5 and block sampling at 1 ms
while profiling is active. This adds runtime overhead, so disable profiling
after collecting the needed captures. For a one-off shell launch, use `env`
because the project config prefix contains a hyphen and cannot be written as a
normal shell assignment:

```sh
env 'gowa-ui_PROFILING__ENABLED=true' \
    'gowa-ui_PROFILING__ADDRESS=127.0.0.1:6060' \
    ./gowa-ui server
```

You can also set these keys in `config.toml` and restart the service.

## Reach the listener

For a direct host deployment, create an SSH tunnel from your workstation:

```sh
ssh -N -L 6060:127.0.0.1:6060 user@server
```

Keep the tunnel open, then use `http://127.0.0.1:6060` on your workstation.
Do not publish this port through a public reverse proxy or firewall rule.

In Docker bridge networking, `127.0.0.1` is the container's loopback, not the
Docker host. Capture a profile from inside the application container, then
copy it out. For example, when `curl` is available in the container:

```sh
docker exec <container> curl -fsS \
  'http://127.0.0.1:6060/debug/pprof/heap' -o /tmp/heap.pb.gz
docker cp <container>:/tmp/heap.pb.gz ./heap.pb.gz
```

Use the same approach for the other paths below. In Kubernetes, containers in
the same pod share loopback; an ephemeral diagnostic container in that pod can
capture the profile. A standard SSH local forward works directly when the
application shares the host network namespace; Docker bridge containers need
an in-container capture or a deliberately configured local-only relay.

## Capture profiles

These examples assume the listener is reachable at `127.0.0.1:6060`. Save
captures to a protected directory because profile contents can be sensitive.

```sh
# 30-second CPU sample
curl -fsS 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30' -o cpu.pb.gz

# Heap allocation snapshot
curl -fsS 'http://127.0.0.1:6060/debug/pprof/heap' -o heap.pb.gz

# Goroutine stacks and wait states
curl -fsS 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=2' -o goroutines.txt

# Mutex and blocking contention; runtime sampling is enabled while this
# profiling listener is active.
curl -fsS 'http://127.0.0.1:6060/debug/pprof/mutex' -o mutex.pb.gz
curl -fsS 'http://127.0.0.1:6060/debug/pprof/block' -o block.pb.gz
```

Inspect a profile with Go's pprof tool:

```sh
go tool pprof cpu.pb.gz
go tool pprof heap.pb.gz
go tool pprof mutex.pb.gz
go tool pprof block.pb.gz
```

At the pprof prompt, `top` shows the largest contributors and `web` opens a
call graph when Graphviz is installed. Goroutine text captures can be reviewed
directly or fetched as a pprof profile without `debug=2` for aggregation.

## Compare before and after

Capture comparable profiles under the same workload and duration before and
after a change. For example:

```sh
go tool pprof -base cpu-before.pb.gz cpu-after.pb.gz
go tool pprof -base heap-before.pb.gz heap-after.pb.gz
go tool pprof -base mutex-before.pb.gz mutex-after.pb.gz
```

The resulting view reports the change between the baseline and the later
capture. Compare profiles collected at similar traffic levels; CPU and
contention profiles represent activity over time, while heap is a snapshot.

Profiles can include function names, stack traces, command-line arguments,
allocation labels, and application data present in memory. Treat captures as
confidential operational data: restrict access, avoid attaching them to public
issues, and delete them when the investigation is complete. Disable profiling
in the config and restart the service after the diagnostic window.
