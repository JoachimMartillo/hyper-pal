# Installation

## Dependencies

* go get -u github.com/astaxie/beego
* go get -u github.com/beego/bee
* go get -u github.com/go-sql-driver/mysql
* go get -u github.com/satori/go.uuid

To load all libraries:

* go get ./...

## Compile

* env GOOS=linux GOARCH=amd64 go build -o pal-import

#### DB Helpers

* How to delete `ALL` parsed Tags from system:

```
begin;
delete from TagsInLibraries where uuid in (
    select tag_id from ClassificationsInPal where tag_id is not null
);
delete from ClassificationsInPal;
commit;
```

* How to make script do not wait 15 minutes and start background job immediately:

```
update PalSpaces set proceeded_at = addtime(proceeded_at, '-00:20:00');
```