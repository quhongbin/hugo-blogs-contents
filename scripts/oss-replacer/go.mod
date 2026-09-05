// 独立模块：CI 用的「OSS 链接替换工具」。
//
// 之所以单独一个 module 而不是与 scripts/oss-uploader 共用：
//   - 上传工具是「写」操作（PutObject），替换工具是「读」操作（ListObjects），
//     两者可以给 CI 的 RAM 子账号授予完全不同的权限，分开便于各自演进；
//   - 独立 module 对已经跑通的上传工具零改动，风险最低。
// 代价是依赖版本需要两边一起升级（目前两处都是 SDK v1.6.0）。
module oss-replacer

go 1.24

require github.com/aliyun/alibabacloud-oss-go-sdk-v2 v1.6.0

require golang.org/x/time v0.4.0 // indirect
