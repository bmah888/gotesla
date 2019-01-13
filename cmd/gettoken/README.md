gettoken

Obtain an authentication token from Tesla API servers.

Pass MyTesla account credentials using the -email and -password options.

Returns the Tesla bearer token, which is required for most other API
calls.  To see the entire token JSON structure (including the
expiration and renewal fields), also pass the -json flag.
