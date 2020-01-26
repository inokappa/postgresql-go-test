# postgresql-go-test

## これは

* PostgreSQL の設定を Go の testing パッケージで検証するサンプル
* テストどんぶりインフラ (Test-Driven Infrastructure なんちゃって

## Build and Run

```sh
$ docker-compose build && docker-compose up -d
$ docker-compose exec go_test richgo test -v
```
