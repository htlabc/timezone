package consensus

import "crypto/md5"

func Hash(data string) []byte {
	Md5Inst := md5.New()
	Md5Inst.Write([]byte(data))
	result := Md5Inst.Sum([]byte(""))
	return result
}
