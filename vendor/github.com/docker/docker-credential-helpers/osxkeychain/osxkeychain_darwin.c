#include "osxkeychain_darwin.h"

char *get_error(OSStatus status) {
  char *buf = malloc(128);
  CFStringRef str = SecCopyErrorMessageString(status, NULL);
  int success = CFStringGetCString(str, buf, 128, kCFStringEncodingUTF8);
  if (!success) {
    strncpy(buf, "Unknown error", 128);
  }
  return buf;
}

char *keychain_add(struct Server *server, char *username, char *secret) {
  OSStatus status = SecKeychainAddInternetPassword(
    NULL,
    strlen(server->host), server->host,
    0, NULL,
    strlen(username), username,
    strlen(server->path), server->path,
    server->port,
    server->proto,
    kSecAuthenticationTypeDefault,
    strlen(secret), secret,
    NULL
  );
  if (status) {
    return get_error(status);
  }
  return NULL;
}

char *keychain_get(struct Server *server, unsigned int *username_l, char **username, unsigned int *secret_l, char **secret) {
  char *tmp;
  SecKeychainItemRef item;

  OSStatus status = SecKeychainFindInternetPassword(
    NULL,
    strlen(server->host), server->host,
    0, NULL,
    0, NULL,
    strlen(server->path), server->path,
    server->port,
    server->proto,
    kSecAuthenticationTypeDefault,
    secret_l, (void **)&tmp,
    &item);

  if (status) {
    return get_error(status);
  }

  *secret = strdup(tmp);
  SecKeychainItemFreeContent(NULL, tmp);

  SecKeychainAttributeList list;
  SecKeychainAttribute attr;

  list.count = 1;
  list.attr = &attr;
  attr.tag = kSecAccountItemAttr;

  status = SecKeychainItemCopyContent(item, NULL, &list, NULL, NULL);
  if (status) {
    return get_error(status);
  }

  *username = strdup(attr.data);
  *username_l = attr.length;
  SecKeychainItemFreeContent(&list, NULL);

  return NULL;
}

char *keychain_delete(struct Server *server) {
  SecKeychainItemRef item;

  OSStatus status = SecKeychainFindInternetPassword(
    NULL,
    strlen(server->host), server->host,
    0, NULL,
    0, NULL,
    strlen(server->path), server->path,
    server->port,
    server->proto,
    kSecAuthenticationTypeDefault,
    0, NULL,
    &item);

  if (status) {
    return get_error(status);
  }

  status = SecKeychainItemDelete(item);
  if (status) {
    return get_error(status);
  }
  return NULL;
}
