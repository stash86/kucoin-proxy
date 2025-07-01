# Freqtrade OPS

## Local

```shell
git clone https://github.com/stash86/kucoin-proxy.git
make build
./kucoin-proxy -port 8080 -verbose 1
```

### config.json

```json
{
  "exchange": {
    "name": "kucoin",
    "key": "",
    "secret": "",
    "ccxt_config": {
      "enableRateLimit": false,
      "timeout": 60000,
      "urls": {
        "api": {
          "public": "http://127.0.0.1:8080/kucoin",
          "private": "http://127.0.0.1:8080/kucoin"
        }
      }
    },
    "ccxt_async_config": {
      "enableRateLimit": false,
      "timeout": 60000
    }
  }
}
```

## Docker (suggested way)

### Use different tags for different platforms e.g. - latest-amd64, latest-arm-v6, latest-arm-v7, latest-arm64

```shell
docker run --restart=always -p 127.0.0.1:8080:8080 --name kucoin-proxy -d stash86/kucoin-proxy:latest-amd64
```

### config.json for docker usage

```json
{
  "exchange": {
    "name": "kucoin",
    "key": "",
    "secret": "",
    "ccxt_config": {
      "enableRateLimit": false,
      "timeout": 60000,
      "urls": {
        "api": {
          "public": "http://127.0.0.1:8080/kucoin",
          "private": "http://127.0.0.1:8080/kucoin"
        }
      }
    },
    "ccxt_async_config": {
      "enableRateLimit": false,
      "timeout": 60000
    }
  }
}
```

## Docker-compose (best way)

### Use different tags for different platforms e.g. - latest-amd64, latest-arm-v6, latest-arm-v7, latest-arm64 for compose

See example - [docker-compose.yml](freqtrade-docker-compose.yml)

```yaml
  kucoin-proxy:
    image: stash86/kucoin-proxy:latest-amd64
    restart: unless-stopped
    container_name: kucoin-proxy
    command: -verbose 1
```

### config.json for docker-compose

```json
{
  "exchange": {
    "name": "kucoin",
    "key": "",
    "secret": "",
    "ccxt_config": {
      "enableRateLimit": false,
      "timeout": 60000,
      "urls": {
        "api": {
          "public": "http://kucoin-proxy:8080/kucoin",
          "private": "http://kucoin-proxy:8080/kucoin"
        }
      }
    },
    "ccxt_async_config": {
      "enableRateLimit": false,
      "timeout": 60000
    }
  }
}
```
