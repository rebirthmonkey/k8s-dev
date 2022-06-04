# code-generator

## Lab
注意：go.mod 必须在 rebirthmonkey 目录下，并且 module 名必须与目录名相同（github.com/rebirthmonkey)

- generate code
```shell
cd 42_code-generator
 
../../../../code-generator/generate-groups.sh all github.com/rebirthmonkey/pkg/generated \
github.com/rebirthmonkey/pkg/apis \
wukong.com:v1 --go-header-file ./boilerplate.go.txt --output-base ../../
```

