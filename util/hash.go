package consensus

import (
	"crypto/md5"
	"encoding/hex"
)

func Hash(data []byte) string {
	//给哈希算法添加数据
	res := md5.Sum(data) //返回值：[Size]byte 数组
	/*//方法1：
	result=fmt.Sprintf("%x",res)   //通过fmt.Sprintf()方法格式化数据
	*/
	//方法2：
	result := hex.EncodeToString(res[:]) //对应的参数为：切片，需要将数组转换为切片。
	return result
}
