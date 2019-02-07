checktoken
==========

Do various operations on the cached authenticaation token, which must
be generated/saved with the gettoken command.  The actual function of
this program depends on a single command word given after any flags
such as `-json`.

check
-----
Check the timestamp validity of the cached token.  Process return
code 0 if the cached token is valid according to the timestamps, 1
otherwise.

clear
-----
Clear the cache.  This operation does not invalidate the token, so if
the bearer token exists anywhere else (in a file, etc.) it can still
be used for vehicle access.

print
-----
Print the authentication bearer token, or the complete token structure
if the `-json` flag is given.
