# build cache 需求

1. 工程目录下所有文件生成对应的 md5
2. md5 文件由多行构成，每行格式为 `pathname digest`
3. digest 文件不存在就 skip，直接生成新的
4. 多 goroutine 并发生成 md5( may skip，并发读 disk 性能提升可能不大 )
5. 最好再实现 copy 功能（把 copy 和生成 md5 逻辑放一块）