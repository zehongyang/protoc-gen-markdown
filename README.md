# protoc-gen-markdown
将protobuf文档生成markdown文档
## install

```
go install github.com/zehongyang/protoc-gen-markdown
```

## generate markdown

```
protoc --markdown_out=. hello.proto
protoc --markdown_out=router=routes:. hello.proto
说明
protoc --markdown_out=params1=values1:outputdir hello.proto
```
