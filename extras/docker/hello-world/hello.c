#include <unistd.h>

const char message[] =
  "\n"
  "Hello from @willdurand!\n"
  "\n"
  "This message shows that your installation appears to be working correctly\n"
  "(but that might be a lie because this is bleeding edge technology).\n"
  "\n"
  "To generate this message, Yaman took the following steps:\n"
  " 1. Yaman pulled the \"willdurand/hello-world\" image from the Docker Hub.\n"
  " 2. Yaman created a new container from that image which runs the executable\n"
  "    that produces the output you are currently reading. Under the hood,\n"
  "    a \"shim\" named Yacs has been executed. This is the tool responsible\n"
  "    for monitoring the container (which was created by a third tool: Yacr,\n"
  "    an \"OCI runtime\").\n"
  " 3. Yaman connected to the container output (via the shim), which sent it\n"
  "    to your terminal. Amazing, right?\n"
  "\n"
  "To try something more ambitious, you can run an Alpine container with:\n"
  " $ sudo yaman c run -it docker.io/library/alpine sh\n"
  "\n"
  "That's basically it because this is a learning project :D\n"
  "\n"
  "For more examples and ideas, visit:\n"
  " https://github.com/willdurand/containers\n"
  "\n";

int main() {
  write(1, message, sizeof(message) - 1);

  return 0;
}
