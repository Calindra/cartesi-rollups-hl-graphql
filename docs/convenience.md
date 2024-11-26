# Convenience

How to enable the node v2 synchronization

```bash
go build . && \
    rm -rf db.sqlite3 && \
    ./nonodo --raw-enabled -d \
        --graphile-disable-sync \
        --sqlite-file db.sqlite3
```

Run without cleaning the sqlite file

```bash
go build . && \
    ./nonodo --raw-enabled -d \
        --graphile-disable-sync \
        --sqlite-file db.sqlite3
```

## Logbook

### 2024-10-20

PostGraphile has been deprecated. We now read directly from the Node v2 database.

> ![Convenience Layer](convenience-diagram.png)
> In orange is the GraphQL federation for future version migrations.

Node v2 will handle the voucher execution flag.
