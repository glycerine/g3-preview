#include <emscripten/console.h>
#include <stdint.h>
#include <sys/resource.h>

namespace {

const char *resourceName(int resource) {
  switch (resource) {
  case RLIMIT_CPU:
    return "RLIMIT_CPU";
  case RLIMIT_FSIZE:
    return "RLIMIT_FSIZE";
  case RLIMIT_DATA:
    return "RLIMIT_DATA";
  case RLIMIT_STACK:
    return "RLIMIT_STACK";
  case RLIMIT_CORE:
    return "RLIMIT_CORE";
#ifdef RLIMIT_RSS
  case RLIMIT_RSS:
    return "RLIMIT_RSS";
#endif
#ifdef RLIMIT_NPROC
  case RLIMIT_NPROC:
    return "RLIMIT_NPROC";
#endif
#ifdef RLIMIT_NOFILE
  case RLIMIT_NOFILE:
    return "RLIMIT_NOFILE";
#endif
#ifdef RLIMIT_MEMLOCK
  case RLIMIT_MEMLOCK:
    return "RLIMIT_MEMLOCK";
#endif
#ifdef RLIMIT_AS
  case RLIMIT_AS:
    return "RLIMIT_AS";
#endif
#ifdef RLIMIT_LOCKS
  case RLIMIT_LOCKS:
    return "RLIMIT_LOCKS";
#endif
#ifdef RLIMIT_SIGPENDING
  case RLIMIT_SIGPENDING:
    return "RLIMIT_SIGPENDING";
#endif
#ifdef RLIMIT_MSGQUEUE
  case RLIMIT_MSGQUEUE:
    return "RLIMIT_MSGQUEUE";
#endif
#ifdef RLIMIT_NICE
  case RLIMIT_NICE:
    return "RLIMIT_NICE";
#endif
#ifdef RLIMIT_RTPRIO
  case RLIMIT_RTPRIO:
    return "RLIMIT_RTPRIO";
#endif
#ifdef RLIMIT_RTTIME
  case RLIMIT_RTTIME:
    return "RLIMIT_RTTIME";
#endif
  default:
    return "UNKNOWN";
  }
}

} // namespace

extern "C" int __syscall_prlimit64(int pid, int resource, intptr_t new_limit,
                                   intptr_t old_limit) {
  const auto *requested = reinterpret_cast<const struct rlimit *>(new_limit);
  auto *current = reinterpret_cast<struct rlimit *>(old_limit);

  if (requested) {
    emscripten_errf(
        "[z3 syscall] prlimit64(pid=%d, resource=%s(%d), newLimitPtr=0x%lx "
        "{cur=%llu,max=%llu}, oldLimitPtr=0x%lx)\n",
        pid, resourceName(resource), resource, static_cast<unsigned long>(new_limit),
        static_cast<unsigned long long>(requested->rlim_cur),
        static_cast<unsigned long long>(requested->rlim_max),
        static_cast<unsigned long>(old_limit));
  } else {
    emscripten_errf(
        "[z3 syscall] prlimit64(pid=%d, resource=%s(%d), newLimitPtr=0x%lx, "
        "oldLimitPtr=0x%lx)\n",
        pid, resourceName(resource), resource, static_cast<unsigned long>(new_limit),
        static_cast<unsigned long>(old_limit));
  }

  if (current) {
    current->rlim_cur = RLIM_INFINITY;
    current->rlim_max = RLIM_INFINITY;
    emscripten_errf(
        "[z3 syscall] prlimit64 wrote old limit {cur=%llu,max=%llu}\n",
        static_cast<unsigned long long>(current->rlim_cur),
        static_cast<unsigned long long>(current->rlim_max));
  }

  return 0;
}
