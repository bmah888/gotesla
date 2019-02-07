gettoken
========

Obtains an authentication token from Tesla API servers.

Pass MyTesla account credentials using the `-email` and `-password`
options.

The authentication token, which is passed as a parameter in Tesla API
calls, is saved in the file ~/.gotesla.cache.  This is the place where
other gotesla utilities will expect to find an authentication token;
with the exception of gettoken they generally do not have the means to
generate a new token on their own.

This utility also prints the Tesla bearer token.  To see the entire
token JSON structure (including the expiration and renewal fields),
also pass the `-json` flag.  After the token has been generated, the
checktoken utility can also be used to examine the cached token.
