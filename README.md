# Consul Connect Router
This project is a simple HTTP based API gateway for Consul Connect

The router can be run at the edge of a system and send upstream requests to connect proxies.

NOTE: Code is kinda janky while building proof of concept  

## Running

Upstreams are configured using the --upstream flag which takes is a string in the form of `[service]#[path]`  

**e.g.**  
The following example would route all requests received at the path (including subpaths) `/api` to the `api` service.  Requests received at the path `/` would be routed to the `frontend` service.

```bash
connect-router --upstream "service=api#path=/api" --upstream "service=frontend#path=/"
```
