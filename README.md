# P2P HTTP Relayer

This is an implementation of a Go LibP2P Node using the [LibP2P HTTP Spec](https://github.com/libp2p/specs/tree/master/http) that relays HTTP requests from clients  to HTTP services, both local and remote relative to this server. For now it simply handles transparent proxying, would like to consider TLS tunneling too*.

It is using a fork of go-libp2p that has a modification to the WebRTC transport package that allows me to pass a custom cert, for multiaddress persistance, when creating the libp2p node.

This is done with `replace github.com/libp2p/go-libp2p => github.com/AustinFoss/go-libp2p v0.0.4` in `go.mod`

I created this initially for my project called SENSR and so it has the proxy URLs currently hardcoded. Priority TODO is to seperate this out and pass accepted endpoints via a `.env` file so that others can use it more easily for any HTTP service.

## Usage

For use with my current SENSR project that interacts with an Ethereum RPC endpoint from Alchemy, first duplicate the `.env.tmp` file to a `.env`, and add your own ALCHEMY_API key. Then install dependecies and run the relay.

```bash
go mod tidy
go run src/main.go
```

It will store a certificate, a file with the multiddresses the LibP2P node is listening on, and a LibP2P private key to the `tmp` directory. Backup elsewhere if you like.

## Testing

If you want to test this server for your own LibP2P uses modify the appropriate code in `src/utils/p2phttp/p2phttp.go` to relay to our test http server that we can start with:

```bash
go run testing/http-server.go --port 43115 --message "foo bar"
```

Above means that if the libp2p server is modified to point to `localhost:43115/hello` it will respond with `"foo bar"`. This will all likely be moved to arguements in a `.env` file so that `src/main.go` doesn't need to be edited and simplying CLI commands.