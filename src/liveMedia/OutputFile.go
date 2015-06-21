package liveMedia

import "os"

func OpenOutputFile(fileName string) *os.File {
    fid, err := os.Open(fileName)
    if err != nil {
        return nil
    }
    return fid
}

func CloseOutputFile(fid *os.File) {
    defer fid.Close()
}
