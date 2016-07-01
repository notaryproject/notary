#include <Security/Security.h>

struct Server {
  SecProtocolType proto;
  char *host;
  char *path;
  unsigned int port;
};

char *keychain_add(struct Server *server, char *username, char *secret);
char *keychain_get(struct Server *server, unsigned int *username_l, char **username, unsigned int *secret_l, char **secret);
char *keychain_delete(struct Server *server);
