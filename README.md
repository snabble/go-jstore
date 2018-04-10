
go-jstore
================

Golang interaface library for storage of json data.

See the [Testcode](https://github.com/snabble/go-jstore/blob/master/elastic/es_store_test.go#L23) for an example.


Local testing
-------------------
Running the tests requires a running elasticsearch.

```
sudo sysctl -w vm.max_map_count=262144
docker run --rm -p 9200:9200 docker.elastic.co/elasticsearch/elasticsearch:6.0.1
```
