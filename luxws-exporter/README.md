# luxws-exporter

A [Prometheus exporter][promexporter] using the `Lux_WS` protocol to retrieve
informational values from Luxtronik 2.x heat pump controllers manufactured
and/or deployed by the following companies:

* Alpha Innotec
* NIBE
* Novelan
* possibly other companies and/or brands


## Language support

The exporter must know which language the controller interface is using. See
the [`luxwslang` package](../luxwslang/) for implemented languages (includes
English and German). Other languages are easily added by defining a few
strings.


## Timezone

In order to parse timestamps (e.g. of the most recent error) it's necessary for
the exporter to know the timezone used by the controller. By default the
system-local timezone is used.


## Usage

Run `luxws-exporter -help` for a usage description. Example:

```
./luxws-exporter -controller.address=192.0.2.1:8214 -controller.language=en \
  -controller.timezone=Europe/Berlin -web.listen-address=127.0.0.1:8000
```

Retrieve all values:

```
curl http://127.0.0.1:8000/metrics
```


[promexporter]: https://prometheus.io/docs/instrumenting/exporters/
