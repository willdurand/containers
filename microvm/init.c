#include <errno.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/reboot.h>
#include <sys/stat.h>
#include <termios.h>
#include <unistd.h>

static const char *MV_ENV_VARS[] = {"MV_INIT", "MV_HOSTNAME", "MV_DEBUG",
                                    "MV_TTY", NULL};
static const char *BIN_SH = "/bin/sh";

static void pr_debug(const char *fmt, ...) {
  if (strcmp(getenv("MV_DEBUG"), "1") != 0) {
    return;
  }

  printf("init: ");

  va_list arg;
  va_start(arg, fmt);
  vprintf(fmt, arg);
  va_end(arg);

  printf("\n");
}

static void cleanup_env() {
  const char **env_var = MV_ENV_VARS;
  while (*env_var != NULL) {
    unsetenv(*env_var);
    env_var++;
  }
}

int main(int argc, char *argv[]) {
  if (mkdir("/proc", 0555) != 0 && errno != EEXIST) {
    perror("mkdir: /proc");
    return 1;
  }

  if (mount("proc", "/proc", "proc", 0, NULL) != 0) {
    perror("mount: /proc");
    return 1;
  }

  if (mkdir("/dev", 0755) != 0 && errno != EEXIST) {
    perror("mkdir: /dev/pts");
    return 1;
  }

  if (mkdir("/dev/pts", 0620) != 0 && errno != EEXIST) {
    perror("mkdir: /dev/pts");
    return 1;
  }

  if (mount("devpts", "/dev/pts", "devpts", MS_NOSUID | MS_NOEXEC, NULL) != 0) {
    perror("mount: /dev/pts");
    return 1;
  }

  if (mkdir("/dev/shm", 0777) != 0 && errno != EEXIST) {
    perror("mkdir: /dev/shm");
    return 1;
  }

  if (mount("shm", "/dev/shm", "tmpfs", MS_NOSUID | MS_NOEXEC | MS_NODEV,
            NULL) != 0) {
    perror("mount /dev/shm");
    return 1;
  }

  char *hostname = getenv("MV_HOSTNAME");
  if (hostname) {
    pr_debug("sethostname: %s", hostname);
    sethostname(hostname, strlen(hostname));
  }

  char *init = getenv("MV_INIT");
  if (!init) {
    init = (char *)BIN_SH;
  }
  argv[0] = init;
  pr_debug("execvp: argc=%d argv0=%s", argc, argv[0]);

  if (strcmp(getenv("MV_TTY"), "1") == 0) {
    setsid();

    int fd = open("/dev/hvc0", O_RDWR);
    if (fd < 0) {
      perror("open: /dev/hvc0");
      return 1;
    }
    dup2(fd, 0);
    dup2(fd, 1);
    dup2(fd, 2);
    while (fd > 2) {
      close(fd--);
    }

    // This should fix the following error:
    //
    //   /bin/sh: can't access tty; job control turned off
    //
    ioctl(0, TIOCSCTTY, 1);
  } else {
    // Disable ECHO
    struct termios term;
    tcgetattr(0, &term);
    term.c_lflag &= ~ECHO;
    tcsetattr(0, 0, &term);

    printf("init: ready\n");
  }

  cleanup_env();

  return execvp(argv[0], argv);
}
