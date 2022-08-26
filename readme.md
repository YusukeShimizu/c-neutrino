# Warning
__The plugin is in alpha state__  
__Only available on bitcoin testnet3__

# description
c-neurino is the c-lightning bitcoin backend plugin depends on LND neutrino mode.  
c-lightning can be used without bitcoin core.  
Neutrino is bitcoin's light client to protect privacy and minimize overhead.

## build
Install `go 1.19`, and cross compile for your env.
```sh
env GOOS=linux GOARCH=amd64 go build
```
## Run
* Disable `bcli`, the default bitcoin backend
* Launch LND(`v0.15.1-branch-rc1` or upper) in neutrino mode (RPC must be enabled)
* Register c-neutrino as a plugin


```sh
lightningd --testnet --disable-plugin bcli \
    --plugin <path-to-c-neutrino> \
    --tls-cert-path <path-to-tls-cert-path> \
    --macaroon-path <path-to-macaroon-path>  \
    --grpc-dial <lnd-grpc-port>
```

Register address to get target transaction by neutrino.
```sh
lncli --network=testnet wallet accounts import-pubkey <pubkey of core lightning> p2wkh
```