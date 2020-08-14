package mp4

import (
	"encoding/binary"
	"io"
	"io/ioutil"
)

//AVCDecConfRec - AVCDecoderConfigurationRecord
type AVCDecConfRec struct {
	AVCProfileIndication byte
	ProfileCompatibility byte
	AVCLevelIndication   byte
	SPSnalus             [][]byte
	PPSnalus             [][]byte
}

// CreateAVCDecConfRec - Create an AVCDecConfRec based on SPS and PPS
func CreateAVCDecConfRec(spsNALU []byte, ppsNALUs [][]byte) AVCDecConfRec {
	sps, err := ParseSPSNALUnit(spsNALU)
	if err != nil {
		panic("Could not parse SPS")
	}

	return AVCDecConfRec{
		AVCProfileIndication: byte(sps.Profile),
		ProfileCompatibility: byte(sps.ProfileCompatibility),
		AVCLevelIndication:   byte(sps.Level),
		SPSnalus:             [][]byte{spsNALU},
		PPSnalus:             ppsNALUs,
	}
}

// DecodeAVCDecConfRec - decode an AVCDecConfRec
func DecodeAVCDecConfRec(r io.Reader) (AVCDecConfRec, error) {

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return AVCDecConfRec{}, err
	}
	//configurationVersion := data[0]   // Should be 1
	AVCProfileIndication := data[1]
	ProfileCompatibility := data[2]
	AVCLevelIndication := data[3]
	LengthSizeMinus1 := data[4] & 0x03 // The first 5 bits are 1
	if LengthSizeMinus1 != 0x3 {
		panic("Can only handle 4byte NAL length size")
	}
	numSPS := data[5] & 0x1f // 5 bits following 3 reserved bits
	pos := 6
	spsNALUs := make([][]byte, 0, 1)
	for i := 0; i < int(numSPS); i++ {
		nalLength := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2
		spsNALUs = append(spsNALUs, data[pos:pos+nalLength])
		pos += nalLength
	}
	ppsNALUs := make([][]byte, 0, 1)
	numPPS := data[pos]
	pos++
	for i := 0; i < int(numPPS); i++ {
		nalLength := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2
		ppsNALUs = append(ppsNALUs, data[pos:pos+nalLength])
		pos += nalLength
	}

	return AVCDecConfRec{
		AVCProfileIndication: AVCProfileIndication,
		ProfileCompatibility: ProfileCompatibility,
		AVCLevelIndication:   AVCLevelIndication,
		SPSnalus:             spsNALUs,
		PPSnalus:             ppsNALUs,
	}, nil
}

// Encode - write an AVCDecConfRec to w
func (a *AVCDecConfRec) Encode(w io.Writer) error {
	var err error
	write := func(b byte) {
		if err != nil {
			return
		}
		err = binary.Write(w, binary.BigEndian, b)
	}

	var configurationVersion byte = 1
	var ffByte byte = 0xff
	write(configurationVersion)
	write(a.AVCProfileIndication)
	write(a.ProfileCompatibility)
	write(a.AVCLevelIndication)
	write(ffByte) // Set length to 4
	if err != nil {
		return err
	}

	var nrSPS byte = byte(len(a.SPSnalus)) | 0xe0 // Added reserved 3 bits
	err = binary.Write(w, binary.BigEndian, nrSPS)
	if err != nil {
		return err
	}
	for _, sps := range a.SPSnalus {
		var length uint16 = uint16(len(sps))
		err = binary.Write(w, binary.BigEndian, length)
		if err != nil {
			return err
		}
		_, err = w.Write(sps)
		if err != nil {
			return err
		}
	}
	var nrPPS byte = byte(len(a.PPSnalus))
	err = binary.Write(w, binary.BigEndian, nrPPS)
	if err != nil {
		return err
	}
	for _, pps := range a.PPSnalus {
		var length uint16 = uint16(len(pps))
		err = binary.Write(w, binary.BigEndian, length)
		if err != nil {
			return err
		}
		_, err = w.Write(pps)
		if err != nil {
			return err
		}
	}

	return nil
}
