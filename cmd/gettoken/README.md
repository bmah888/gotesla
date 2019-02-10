gettoken
========

Obtains an authentication token from Tesla API servers.

There are two modes of operation.  The first is a straightforward
login, using an owner's MyTesla account credentials.  Pass these using
the `-email` and `-password` options.  Tokens are valid for about 45
days.

The other mode of operation is to refresh an existing token, before it
expires.  This allows a user's login session to extend much longer
than the 45-day lifetime of a token.  To refresh a token, pass the
`-refresh` command-line flag.

The authentication token, which is passed as a parameter in Tesla API
calls, is saved in the file ~/.gotesla.cache.  This is the place where
other gotesla utilities will expect to find an authentication token;
with the exception of gettoken they generally do not have the means to
generate a new token on their own.

This utility also prints the Tesla bearer token.  To see the entire
token JSON structure (including the expiration and renewal fields),
also pass the `-json` flag.  After the token has been generated, the
checktoken utility can also be used to examine the cached token.
