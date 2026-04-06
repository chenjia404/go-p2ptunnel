package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/polydawn/refmt/json"
)

func CheckGithubVersion(Version string) {
	githubName := "go-p2ptunnel"
	githubPath := "chenjia404/go-p2ptunnel"

	archivesFormat := "tar.gz"
	if runtime.GOOS == "windows" {
		archivesFormat = "zip"
	}

	r, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubPath))
	if err != nil {
		return
	}
	b, err := io.ReadAll(r.Body)
	var v interface{}
	err = json.Unmarshal(b, &v)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := v.(map[string]interface{})

	githubVerion := fmt.Sprintf("%s", data["tag_name"])
	githubVerion = strings.Replace(githubVerion, "v", "", 1)
	if compareVersion(githubVerion, Version) > 0 {
		fmt.Println("GitHub版本更高")
	} else {
		fmt.Println("不需要升级")
		return
	}

	githubPublishedTime, _ := time.ParseInLocation("2006-01-02T15:04:05Z", fmt.Sprintf("%s", data["published_at"]), time.Local)
	if time.Now().Sub(githubPublishedTime) < (time.Second * 3600) {
		fmt.Println("更新时间不足1个小时，延迟更新")
		return
	}
	updateFileUrl := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s_%s_%s_%s.%s", githubPath, githubVerion, githubName, githubVerion, runtime.GOOS, runtime.GOARCH, archivesFormat)
	// Get the data
	resp, err := http.Get(updateFileUrl)
	if err != nil {
		fmt.Println(err)
		return
	}

	if resp.StatusCode == 404 {
		fmt.Println("文件不存在，404错误" + updateFileUrl)
		return
	}
	defer resp.Body.Close()

	// 创建一个文件用于保存
	out, err := os.Create("update." + archivesFormat)
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	// 然后将响应流和文件流对接起来
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	} else {
		fmt.Println("下载最新安装包成功")
	}

	out, err = os.Open("update." + archivesFormat)
	if err != nil {
		fmt.Println(err)
	}
	h := sha512.New()
	if _, err := io.Copy(h, out); err != nil {
		fmt.Println(err)
		return
	}

	fileSha512 := hex.EncodeToString(h.Sum(nil))

	checksumsFileURL := fmt.Sprintf("https://github.com/%s/releases/download/v%s/checksums.txt", githubPath, githubVerion)
	r, err = http.Get(checksumsFileURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	b, err = io.ReadAll(r.Body)
	checksums := string(b)
	if strings.Index(checksums, fileSha512) < 0 {

		fmt.Println("文件sha512错误" + fileSha512)
		return
	}

	ascFileURL := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s_%s_%s_%s.%s.asc", githubPath, githubVerion, githubName, githubVerion, runtime.GOOS, runtime.GOARCH, archivesFormat)
	err = DownloadFile(ascFileURL, fmt.Sprintf("update.%s.asc", archivesFormat))
	if err != nil {
		fmt.Println(err)
		return
	}

	Verify, err := VerifySignature(fmt.Sprintf("update.%s", archivesFormat))
	if err != nil {
		fmt.Println(err)
		return
	}
	if !Verify {
		fmt.Println("gpg签名不通过")
		return
	}

	exeFilename, _ := os.Executable()

	//删除老文件
	if FileExists(path.Base(exeFilename) + ".old") {
		err = os.Remove(path.Base(exeFilename) + ".old")
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	err = os.Rename(path.Base(exeFilename), path.Base(exeFilename)+".old")
	if err != nil {
		fmt.Println(err)
		return
	}

	if archivesFormat == "zip" {

		err = Unzip(fmt.Sprintf("update.%s", archivesFormat), ".")
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		err = UnTarGz(fmt.Sprintf("update.%s", archivesFormat), "")
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println("current version: ", Version)
	fmt.Println("Update to version: ", githubVerion)
	fmt.Println("Ready to restart")
	os.Exit(0)
}

func Unzip(zipPath, dstDir string) error {
	// open zip file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if err := unzipFile(file, dstDir); err != nil {
			return err
		}
	}
	return nil
}

func UnTarGz(tarFile, dest string) error {
	srcFile, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	gr, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		if hdr.Typeflag != tar.TypeDir {

			// Get file information
			fi := hdr.FileInfo()
			filename := dest + hdr.Name
			file, err := createFile(filename)
			if err != nil {
				return err
			}
			io.Copy(file, tr)
			// Set the file permission, so that it can be guaranteed to be the same as the original file permission. If not set, it will be set according to the umask of the current system.
			os.Chmod(fi.Name(), fi.Mode().Perm())
		}
	}
	return nil
}
func createFile(name string) (*os.File, error) {
	if strings.LastIndex(name, "/") >= 0 {

		err := os.MkdirAll(string([]rune(name)[0:strings.LastIndex(name, "/")]), 0755)
		if err != nil {
			return nil, err
		}
	}
	return os.Create(name)
}

func unzipFile(file *zip.File, dstDir string) error {
	// create the directory of file
	filePath := path.Join(dstDir, file.Name)
	if file.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// open the file
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// create the file
	w, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer w.Close()

	w.Chmod(0777)

	// save the decompressed file content
	_, err = io.Copy(w, rc)
	return err
}

func compareVersion(version1 string, version2 string) int {
	var res int
	ver1Strs := strings.Split(version1, ".")
	ver2Strs := strings.Split(version2, ".")
	ver1Len := len(ver1Strs)
	ver2Len := len(ver2Strs)
	verLen := ver1Len
	if len(ver1Strs) < len(ver2Strs) {
		verLen = ver2Len
	}
	for i := 0; i < verLen; i++ {
		var ver1Int, ver2Int int
		if i < ver1Len {
			ver1Int, _ = strconv.Atoi(ver1Strs[i])
		}
		if i < ver2Len {
			ver2Int, _ = strconv.Atoi(ver2Strs[i])
		}
		if ver1Int < ver2Int {
			res = -1
			break
		}
		if ver1Int > ver2Int {
			res = 1
			break
		}
	}
	return res
}

func DownloadFile(url string, dest string) error {
	// Get the data
	resp, err := http.Get(url)

	if resp.StatusCode == 404 {
		fmt.Println("文件不存在，404错误" + url)
		return http.ErrMissingFile
	}
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()

	// 创建一个文件用于保存
	out, err := os.Create(dest)
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	} else {
		fmt.Println("签名文件下载成功")
	}

	return nil
}

var publicKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQINBGRVILgBEACxqkRKodS2Mfxn6GTYvUDaBSgQCjT/GMqmto38buSing9PCXv6
QMWko8Ax7cKVkxEKGD+4T+AD2mLfhpjLBlMOcxqBwuJ4YVsWkHH2TLHc/gU3DL9Y
ajH9Lt8TF+Xin/pBfGdOBXGeKK2Az8RshK5D3w3E89//plL15kaR0BWbVIp6Ne0P
c5D7BNboRuqJGAY+aYEipWAHLZW5M2dD1wgVjUpZRwWv+qIKuQ+hri+fxehFjz3S
8ElwqZu8JQHxcO3b3m3j11x1qfekqRvNf/dxMpuS+ymenAjOmDDlarmSTj9RTzrA
97uYi2meIr5e85yMNk5n8Ks7HOQyQ1K6J7YBodjItO7bp1EE5xSecNsaIT2kBQX3
0+uga0IsZkA6MIC8caWfkMIXrdyLse4XFywCdOGI3BhrA6QV/7ZAXRBs5HtO6SQO
eVfDptZ0VCvmWG8v6d5mBJ6081FylHEoDYXfJVwgRo71UR334WBpRJZQNV76p383
muUSq05IcwjbAdyol26enqO2s5LRNs7OeISAhQ+u2LV6LJK+G23JKbmIuWD7Rhol
gLDXYukoIlOcY7x++qnqoLT8V1aNFE/4XDAd+/Xq7VdgvKbPZxxEkXj9LMrPBIaS
9/1Nmiq/ni779pnGCFDS7UUFLJvWjEDgWKnZb8MYBdyvq9T9biecJ2oR6wARAQAB
tCJjaGVuamlhNDA0IDxjaGVuamlhYmxvZ0BnbWFpbC5jb20+iQJRBBMBCAA7AhsP
BQsJCAcCAiICBhUKCQgLAgQWAgMBAh4HAheAFiEE4TRiUu1mI2TKN/cWGJvnloM2
naMFAmnTJ9EACgkQGJvnloM2naPcWxAAoUin5Whvd3IwvozT7SllUWHiRABZ4Y9F
3eH/5hionZVLYtaWF6CAMcxlkImsAHuHb3/1NvVkd67yWxgQxn9lNzTe59pF0qTi
A447Z4JYXBeyN8OhoHtoAZfgDf6m4Pvm/nwGnKgBnuKSXCpqBAp8b0mt467QS384
yga9fRpCCiYVsQYGlU3+miuwBpOPAO2/LEVX+IQ9U31YMAYJIlojU3pp3gq/h5+I
RBrtQUsmZGwc/NTyDU83SD+cCbxJaaNEttGhrqBZqlarNZNG8QFfYINCWhni5eiE
hj7Byhix0d27NnBXyy7vf4dd+qYEGIgs5qUu883XyTrHAgw+B/AgC4wsqUJ/t5lC
uMHqWvpJIt9yrAJ1g5e/sQ+kFIKv0rAm6tbmcQDCbId5vcsvBzyYuC2SmEMuuD2K
5TDdTe49OZAtL/LyUs8ZiCqhlrZ2cfWgSfR+QkFUA6hHyyb7wlpQgnuX/5biePvB
H/IwcgzkXqu2yDHu6qSuaBNsgEKEvUjqbzgIrOzvCntYz/dpVk22FodGswSo81TS
hPz7a7UHJbPSvjgVq3X+xp0mhB0YTQ9R+EAlirvq89qKcPbe12EAWMeu3sSJXLIF
S3suotkNu0BGEqO1qXqnsNtV2OYgJrbtWQ5tPIlXXQTFClya3gb3M7+++8azWWdI
1KF1gKEENY+JAlcEEwEIAEECGw8FCwkIBwICIgIGFQoJCAsCBBYCAwECHgcCF4AW
IQThNGJS7WYjZMo39xYYm+eWgzadowUCZKHE6wUJBBQrRgAKCRAYm+eWgzadoyZa
D/9HOwKAS5lOT2r9SXjlkDPC2XNbRBbNGBsq49zwi449D7vGrsNNy/3rLb5CM1SF
m4bJFrdKfO/rJ78oHBlwWQ1e4h+WpR78AMpg7LJBrOOGM1YB6+bZqoXMXhY7oCYH
QenYrHff/m95KNvqrT86cXBIWLO8JBCimMUikp4dwDlfNC82ywYGStKSYA8JVRMg
k69lYMjTRcYybFZkIJJR2Exgl/K8dLL6gRZ7G/07FLn90JD8iEQe7XXjGzxu2oVb
vWa9/phIhjWk6ZZP3sSHvht6Iyprr7ZjAuKbyafVCncC0o9nKx8nNle5PFaE+ayh
T+pnO2AkgJKCRXSQn4/Lp86XwuuLtHmLy1/CNWEGJ5zZ5ZR7NCsm00oxus0EiQ4S
CI2x7PLsewZVv37taSTGdtg+6mG+Bfzrl2b0VYKym614k6QNJy3BDdyex1fmDPXG
L8viMNsF0zQT0iWcPspNgyF7Po2pedUNKRXPIJO/OH5JKlT6rD9hCcMLR3OS1bMx
qoatFBPoiLOOR0S4iAStgWkhwXMWeZXOzrYka7Ez4VqI1bKnoFo0gDnWmmL9NMcA
xIklOc0mpZvUcEtGE77NCrBrCCRtw/yBRY+Y/y6j57mxagjjQrd6pzEihGgNOxM5
W10jm98gafYRhxELuY0cHxkeH6/TLBGWQH5EJ15M2csUWrgzBGSm5QMWCSsGAQQB
2kcPAQEHQGAshEreF6QFo5n4RrQyU/QsOPKjbFAHNR7fX+U/KtgUiQKzBBgBCAAm
AhsCFiEE4TRiUu1mI2TKN/cWGJvnloM2naMFAmhKIlsFCQlJAEAAgXYgBBkWCgAd
FiEE+BfOgVjFV+TJ7XpBp+juoc8oZ/AFAmSm5QMACgkQp+juoc8oZ/DI+AEAqcBg
pgS2Q6cdVv2CGrBnwzYifUgZzsFu5Ve8VUUel5YBAKH3BSNW1WuRVzI6Xq4Beg42
wfhU6Mq9izZALIhJp+oMCRAYm+eWgzadoytTEACLLCYuxDCkKMKhPrZxO2rzIqy/
aNzYAFXHOA6TKqwlAyeyaKt4c5k1d/Zcu492xgrNUqwZeOTCp3In+fl+zklM/0zl
3PNRUGFENarWL9ghqaRRK5HmhMeGOdZ4gYq8eDHGpaIhtGfSpQrpc6/0MNiiG4LK
EIhWqEBTNbr4ZsKd7/QCilDw55a+Y0jyifB4/qiz4yr8qHTg+3GnLoR6a6C4B4/Z
dD4R7stPy0Pauvi/zHlVI6rRirnzV8kABF2ve5scMhhDQxJxQOSeFfldP8llx9jm
kaYb39YDX+329yyZDXs7uv0/CMIG1r4lLrJVJEchkqsMFDIwugA+ILo96XX2y3iw
9BtdGy/A2wnBZPzqBRqDIL0Ic28Sm9KUj3xIgVi5/r9zLjMsCwzsOdlp+ls5S7DR
AFAwqFdeD+/sTaw0D7lcjOCbXlyehvL8Lf2ginHw4QR0+4S3Rn/31OqXoMxExBaY
LBCtcFCv2Ba9m1ktrRd6AcFFvm0bdYggHdp4m0OpOaLfX/BDtpN1Ioi3G1mOYSf6
d0X0KK6LjsvwXce6UQmq8MtnftgRG6EvKIwbxZVYZIbgxpwNBP9kvxT627ceNMY/
mpsZZAUyGPeZlaQr+YbqjsrQlohevpkH6nQ3QV12LL2+Y/uW4z/pq7uI+TSkZu9B
8/JDAFkpk/PzzmpIBQ==
=cbBx
-----END PGP PUBLIC KEY BLOCK-----`

func VerifySignature(filename string) (bool, error) {
	keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewReader([]byte(publicKey)))
	if err != nil {
		fmt.Println("Read Armored Key Ring: " + err.Error())
		return false, err
	}

	signature, err := os.Open(filename + ".asc")
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	defer signature.Close()

	verificationTarget, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	defer verificationTarget.Close()

	entity, err := openpgp.CheckArmoredDetachedSignature(keyring, verificationTarget, signature, nil)
	if err != nil {
		fmt.Println("Check Detached Signature: " + err.Error())
		return false, err
	}
	if entity.PrimaryKey.KeyIdString() == "189BE79683369DA3" {
		return true, nil
	} else {
		return false, nil
	}

}

func FileExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
