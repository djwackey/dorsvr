package livemedia

import "fmt"
import sys "syscall"

var SPS_MAX_SIZE uint = 1000
var SEI_MAX_SIZE uint = 5000 // larger than the largest possible SEI NAL unit

var numNextSliceHeaderBytesToAnalyze uint = 12

//////// H264VideoStreamParser ////////
type H264VideoStreamParser struct {
	MPEGVideoStreamParser
	outputStartCodeSize        int
	firstByteOfNALUnit         uint
	log2MaxFrameNum            uint
	frameMbsOnlyFlag           bool
	separateColourPlaneFlag    bool
	haveSeenFirstStartCode     bool
	haveSeenFirstByteOfNALUnit bool
}

func newH264VideoStreamParser(inputSource IFramedSource) *H264VideoStreamParser {
	parser := new(H264VideoStreamParser)
	parser.log2MaxFrameNum = 5
	parser.frameMbsOnlyFlag = true
	parser.initMPEGVideoStreamParser(inputSource)
	return parser
}

func (p *H264VideoStreamParser) parse() uint {
	// The stream must start with a 0x00000001
	if !p.haveSeenFirstStartCode {
		for first4Bytes := p.test4Bytes(); first4Bytes != 0x00000001; {
			p.get1Byte()
			p.setParseState()
		}

		// skip this initial code
		p.skipBytes(4)
		p.haveSeenFirstStartCode = true
	}

	if p.outputStartCodeSize > 0 && p.curFrameSize() == 0 && !p.HaveSeenEOF() {
		// Include a start code in the output:
		p.save4Bytes(0x00000001)
	}

	if p.HaveSeenEOF() {
		// We hit EOF the last time that we tried to parse this data, so we know that any remaining unparsed data
		// forms a complete NAL unit, and that there's no 'start code' at the end:
		remainingDataSize := p.TotNumValidBytes() - p.curOffset()
		for remainingDataSize > 0 {
			nextByte := p.get1Byte()
			if !p.haveSeenFirstByteOfNALUnit {
				p.firstByteOfNALUnit = nextByte
				p.haveSeenFirstByteOfNALUnit = true
			}
			p.saveByte(nextByte)
			remainingDataSize--
		}

		p.get1Byte() // forces another read, which will cause EOF to get handled for real this time
		return 0
	} else {
		next4Bytes := p.test4Bytes()
		if !p.haveSeenFirstByteOfNALUnit {
			p.firstByteOfNALUnit = next4Bytes >> 24
			p.haveSeenFirstByteOfNALUnit = true
		}
		for next4Bytes != 0x00000001 && (next4Bytes&0xFFFFFF00) != 0x00000100 {
			// We save at least some of "next4Bytes".
			if next4Bytes&0xFF > 1 {
				// Common case: 0x00000001 or 0x000001 definitely doesn't begin anywhere in "next4Bytes",
				// so we save all of it:
				p.save4Bytes(next4Bytes)
				p.skipBytes(4)
			} else {
				// Save the first byte, and continue testing the rest:
				p.saveByte(next4Bytes >> 24)
				p.skipBytes(1)
			}
			p.setParseState() // ensures forward progress
			next4Bytes = p.test4Bytes()
		}
		// Assert: next4Bytes starts with 0x00000001 or 0x000001,
		// and we've saved all previous bytes (forming a complete NAL unit).
		// Skip over these remaining bytes, up until the start of the next NAL unit:
		if next4Bytes == 0x00000001 {
			p.skipBytes(4)
		} else {
			p.skipBytes(3)
		}
	}

	nalRefIdc := p.firstByteOfNALUnit & 0x60 >> 5
	nalUnitType := p.firstByteOfNALUnit & 0x1F
	p.haveSeenFirstByteOfNALUnit = false // for the next NAL unit that we parse

	switch nalUnitType {
	case 6: // Supplemental enhancement information (SEI)
		// Later, perhaps adjust "fPresentationTime" if we saw a "pic_timing" SEI payload??? #####
		p.analyzeSEIData()
	case 7: // Sequence parameter set
		// First, save a copy of this NAL unit, in case the downstream object wants to see it:
		size := uint(len(p.buffTo) - len(p.startOfFrame) - p.outputStartCodeSize)
		p.usingSource.saveCopyOfSPS(p.startOfFrame[p.outputStartCodeSize:], size)

		// Parse this NAL unit to check whether frame rate information is present:
		spsData := p.analyzeSPSData()
		if spsData.timeScale > 0 && spsData.numUnitsInTick > 0 {
			p.usingSource.frameRate = float64(spsData.timeScale / (2.0 * spsData.numUnitsInTick))
		} else {
		}
	case 8: // Picture parameter set
		// Save a copy of this NAL unit, in case the downstream object wants to see it:
		size := uint(len(p.buffTo) - len(p.startOfFrame) - p.outputStartCodeSize)
		p.usingSource.saveCopyOfPPS(p.startOfFrame[p.outputStartCodeSize:], size)
	}

	p.usingSource.setPresentationTime()

	thisNALUnitEndsAccessUnit := false // until we learn otherwise
	if p.HaveSeenEOF() {
		// There is no next NAL unit, so we assume that this one ends the current 'access unit':
		thisNALUnitEndsAccessUnit = true
	} else {
		isVCL := nalUnitType <= 5 && nalUnitType > 0 // Would need to include type 20 for SVC and MVC #####
		if isVCL {
			var firstByteOfNextNALUnit uint
			//this.testBytes(firstByteOfNextNALUnit, 1)
			nextNalRefIdc := (firstByteOfNextNALUnit & 0x60) >> 5
			nextNalUnitType := firstByteOfNextNALUnit & 0x1F
			if nextNalUnitType >= 6 {
				// The next NAL unit is not a VCL; therefore, we assume that this NAL unit ends the current 'access unit':
				thisNALUnitEndsAccessUnit = true
			} else {
				// The next NAL unit is also a VCL.  We need to examine it a little to figure out if it's a different 'access unit'.
				// (We use many of the criteria described in section 7.4.1.2.4 of the H.264 specification.)
				var idrPicFlag bool
				if nalUnitType == 5 {
					idrPicFlag = true
				}
				var nextIdrPicFlag bool
				if nextNalUnitType == 5 {
					nextIdrPicFlag = true
				}

				if nextIdrPicFlag != idrPicFlag {
					// IdrPicFlag differs in value
					thisNALUnitEndsAccessUnit = true
				} else if nextNalRefIdc != nalRefIdc && nextNalRefIdc*nalRefIdc == 0 {
					// nal_ref_idc differs in value with one of the nal_ref_idc values being equal to 0
					thisNALUnitEndsAccessUnit = true
				} else if (nalUnitType == 1 ||
					nalUnitType == 2 ||
					nalUnitType == 5) && (nextNalUnitType == 1 ||
					nextNalUnitType == 2 ||
					nextNalUnitType == 5) {
					// Both this and the next NAL units begin with a "slice_header".
					// Parse this (for each), to get parameters that we can compare:

					// Current NAL unit's "slice_header":
					thisSliceHeader := p.analyzeSliceHeader(p.startOfFrame[p.outputStartCodeSize:], p.buffTo, nalUnitType)

					// Next NAL unit's "slice_header":
					nextSliceHeaderBytes := make([]byte, numNextSliceHeaderBytesToAnalyze)
					p.testBytes(nextSliceHeaderBytes, numNextSliceHeaderBytesToAnalyze)

					nextSliceHeader := p.analyzeSliceHeader(nextSliceHeaderBytes, nextSliceHeaderBytes, nextNalUnitType)

					if nextSliceHeader.frameNum != thisSliceHeader.frameNum {
						// frameNum differs in value
						thisNALUnitEndsAccessUnit = true
					} else if nextSliceHeader.picParameterSetID != thisSliceHeader.picParameterSetID {
						// picParameterSetID differs in value
						thisNALUnitEndsAccessUnit = true
					} else if nextSliceHeader.fieldPicFlag != thisSliceHeader.fieldPicFlag {
						// fieldPicFlag differs in value
						thisNALUnitEndsAccessUnit = true
					} else if nextSliceHeader.bottomFieldFlag != thisSliceHeader.bottomFieldFlag {
						// bottom_field_flag differs in value
						thisNALUnitEndsAccessUnit = true
					} else if nextIdrPicFlag == true && nextSliceHeader.idrPicID != thisSliceHeader.idrPicID {
						// IdrPicFlag is equal to 1 for both and idr_pic_id differs in value
						// Note: We already know that IdrPicFlag is the same for both.
						thisNALUnitEndsAccessUnit = true
					}
				}
			}
		}
	}

	if thisNALUnitEndsAccessUnit {
		p.usingSource.pictureEndMarker = true
		p.usingSource.pictureCount++

		// Note that the presentation time for the next NAL unit will be different:
		nextPT := p.usingSource.nextPresentationTime // alias
		nextPT = p.usingSource.presentationTime
		nextFraction := nextPT.Usec/1000000.0 + 1/int64(p.usingSource.frameRate)
		nextSecsIncrement := nextFraction
		nextPT.Sec += nextSecsIncrement
		nextPT.Usec = (nextFraction - nextSecsIncrement) * 1000000
	}
	p.setParseState()

	return p.curFrameSize()
}

func (p *H264VideoStreamParser) removeEmulationBytes(nalUnitCopy []byte, maxSize uint) uint {
	nalUnitOrig := p.startOfFrame[p.outputStartCodeSize:]
	var NumBytesInNALunit uint //p.buffTo - nalUnitOrig
	if NumBytesInNALunit > maxSize {
		return 0
	}
	var nalUnitCopySize, i uint
	for i = 0; i < NumBytesInNALunit; i++ {
		if i+2 < NumBytesInNALunit && nalUnitOrig[i] == 0 && nalUnitOrig[i+1] == 0 && nalUnitOrig[i+2] == 3 {
			nalUnitCopy[nalUnitCopySize] = nalUnitOrig[i]
			i++
			nalUnitCopySize++
			nalUnitCopy[nalUnitCopySize] = nalUnitOrig[i]
			i++
			nalUnitCopySize++
		} else {
			nalUnitCopy[nalUnitCopySize] = nalUnitOrig[i]
			nalUnitCopySize++
		}
	}
	return nalUnitCopySize
}

type sliceHeader struct {
	frameNum          uint
	idrPicID          uint
	picParameterSetID uint
	fieldPicFlag      bool
	bottomFieldFlag   bool
}

func (p *H264VideoStreamParser) analyzeSliceHeader(start, end []byte, nalUnitType uint) *sliceHeader {
	totNumBits := uint(8 * (len(end) - len(start)))
	bv := newBitVector(start, 0, totNumBits)

	// Some of the result parameters might not be present in the header; set them to default values:
	header := new(sliceHeader)

	// Note: We assume that there aren't any 'emulation prevention' bytes here to worry about...
	bv.skipBits(8) // forbidden_zero_bit; nal_ref_idc; nal_unit_type
	firstMBInSlice := bv.getExpGolomb()
	sliceType := bv.getExpGolomb()
	fmt.Printf("%d, %d", firstMBInSlice, sliceType)
	header.picParameterSetID = bv.getExpGolomb()

	if p.separateColourPlaneFlag {
		bv.skipBits(2) // colour_plane_id
	}

	header.frameNum = bv.getBits(p.log2MaxFrameNum)
	if !p.frameMbsOnlyFlag {
		header.fieldPicFlag = bv.get1BitBoolean()
		if header.fieldPicFlag {
			header.bottomFieldFlag = bv.get1BitBoolean()
		}
	}

	var idrPicFlag bool
	if nalUnitType == 5 {
		idrPicFlag = true
	}

	if idrPicFlag {
		header.idrPicID = bv.getExpGolomb()
	}
	return header
}

type seqParameterSet struct {
	timeScale          uint
	numUnitsInTick     uint
	fixedFrameRateFlag uint
}

func (p *H264VideoStreamParser) analyzeSPSData() *seqParameterSet {
	// Begin by making a copy of the NAL unit data, removing any 'emulation prevention' bytes:
	sps := make([]byte, SPS_MAX_SIZE)
	spsSize := p.removeEmulationBytes(sps, SPS_MAX_SIZE)

	bv := newBitVector(sps, 0, 8*spsSize)

	bv.skipBits(8) // forbidden_zero_bit; nal_ref_idc; nal_unit_type
	profileIdc := bv.getBits(8)
	constraintSetNFlag := bv.getBits(8) // also "reserved_zero_2bits" at end
	fmt.Println(constraintSetNFlag)
	levelIdc := bv.getBits(8)
	fmt.Println(levelIdc)
	seqParameterSetID := bv.getExpGolomb()
	fmt.Println(seqParameterSetID)
	if profileIdc == 100 ||
		profileIdc == 110 ||
		profileIdc == 122 ||
		profileIdc == 244 ||
		profileIdc == 44 ||
		profileIdc == 83 ||
		profileIdc == 86 ||
		profileIdc == 118 ||
		profileIdc == 128 {
		chromaFormatIdc := bv.getExpGolomb()
		if chromaFormatIdc == 3 {
			eparateColourPlaneFlag := bv.get1BitBoolean()
			fmt.Println(eparateColourPlaneFlag)
		}
	}

	bv.getExpGolomb() // bit_depth_luma_minus8
	bv.getExpGolomb() // bit_depth_chroma_minus8
	bv.skipBits(1)    // qpprime_y_zero_transform_bypass_flag
	seqScalingMatrixPresentFlag := bv.get1Bit()
	if seqScalingMatrixPresentFlag != 0 {
		cond := 12
		var chromaFormatIdc uint
		if chromaFormatIdc != 3 {
			//cond := 8
		}

		for i := 0; i < cond; i++ {
			seqScalingMatrixPresentFlag := bv.get1Bit()
			if seqScalingMatrixPresentFlag != 0 {
				sizeOfScalingList := 24
				if i < 6 {
					sizeOfScalingList = 16
				}
				var lastScale uint = 8
				var nextScale uint = 8
				for j := 0; j < sizeOfScalingList; j++ {
					if nextScale != 0 {
						deltaScale := bv.getExpGolomb()
						nextScale = (lastScale + deltaScale + 256) % 256
					}
					if nextScale != 0 {
						lastScale = nextScale
					}
				}
			}
		}
	}

	log2MaxFrameNumMinus4 := bv.getExpGolomb()
	p.log2MaxFrameNum = log2MaxFrameNumMinus4 + 4
	picOrderCntType := bv.getExpGolomb()
	if picOrderCntType == 0 {
		log2MaxPicOrderCntLsbMinus4 := bv.getExpGolomb()
		fmt.Println(log2MaxPicOrderCntLsbMinus4)
	} else if picOrderCntType == 1 {
		bv.skipBits(1)    // delta_pic_order_always_zero_flag
		bv.getExpGolomb() // offset_for_non_ref_pic
		bv.getExpGolomb() // offset_for_top_to_bottom_field
		numRefFramesInPicOrderCntCycle := bv.getExpGolomb()
		var i uint
		for i = 0; i < numRefFramesInPicOrderCntCycle; i++ {
			bv.getExpGolomb() // offset_for_ref_frame[i]
		}
	}
	maxNumRefFrames := bv.getExpGolomb()
	fmt.Println(maxNumRefFrames)
	gapsInFrameNumValueAllowedFlag := bv.get1Bit()
	fmt.Println(gapsInFrameNumValueAllowedFlag)
	picWidthInMbsMinus1 := bv.getExpGolomb()
	fmt.Println(picWidthInMbsMinus1)
	picHeightInMapUnitsMinus1 := bv.getExpGolomb()
	fmt.Println(picHeightInMapUnitsMinus1)
	frameMbsOnlyFlag := bv.get1BitBoolean()
	if !frameMbsOnlyFlag {
		bv.skipBits(1) // mb_adaptive_frame_field_flag
	}
	bv.skipBits(1) // direct_8x8_inference_flag
	frameCroppingFlag := bv.get1Bit()
	if frameCroppingFlag != 0 {
		bv.getExpGolomb() // frame_crop_left_offset
		bv.getExpGolomb() // frame_crop_right_offset
		bv.getExpGolomb() // frame_crop_top_offset
		bv.getExpGolomb() // frame_crop_bottom_offset
	}

	spsData := new(seqParameterSet)

	vuiParametersPresentFlag := bv.get1Bit()
	if vuiParametersPresentFlag != 0 {
		spsData = p.analyzeVUIParameters(bv)
	}

	return spsData
}

func (p *H264VideoStreamParser) analyzeSEIData() {
	// Begin by making a copy of the NAL unit data, removing any 'emulation prevention' bytes:
	sei := make([]byte, SEI_MAX_SIZE)
	seiSize := p.removeEmulationBytes(sei, SEI_MAX_SIZE)

	var j uint = 1 // skip the initial byte (forbidden_zero_bit; nal_ref_idc; nal_unit_type); we've already seen it
	for j < seiSize {
		var payloadType uint
		for sei[j] == 255 && j < seiSize {
			j++
			payloadType += uint(sei[j])
		}
		if j >= seiSize {
			break
		}

		var payloadSize uint
		for sei[j] == 255 && j < seiSize {
			j++
			payloadSize += uint(sei[j])
		}
		if j >= seiSize {
			break
		}

		j += payloadSize
	}
}

func (p *H264VideoStreamParser) analyzeVUIParameters(bv *BitVector) *seqParameterSet {
	aspectRatioInfoPresentFlag := bv.get1Bit()
	if aspectRatioInfoPresentFlag != 0 {
		aspectRatioIdc := bv.getBits(8)
		if aspectRatioIdc == 255 /*Extended_SAR*/ {
			bv.skipBits(32) // sar_width; sar_height
		}
	}
	overscanInfoPresentFlag := bv.get1Bit()
	if overscanInfoPresentFlag != 0 {
		bv.skipBits(1) // overscanInfoPresentFlag
	}
	videoSignalTypePresentFlag := bv.get1Bit()
	if videoSignalTypePresentFlag != 0 {
		bv.skipBits(4) // video_format; video_full_range_flag
		colourDescriptionPresentFlag := bv.get1Bit()
		if colourDescriptionPresentFlag != 0 {
			bv.skipBits(24) // colour_primaries; transfer_characteristics; matrix_coefficients
		}
	}
	chroma_loc_info_present_flag := bv.get1Bit()
	if chroma_loc_info_present_flag != 0 {
		bv.getExpGolomb() // chroma_sample_loc_type_top_field
		bv.getExpGolomb() // chroma_sample_loc_type_bottom_field
	}

	spsData := new(seqParameterSet)

	timingInfoPresentFlag := bv.get1Bit()
	if timingInfoPresentFlag != 0 {
		spsData.numUnitsInTick = bv.getBits(32)
		spsData.timeScale = bv.getBits(32)
		spsData.fixedFrameRateFlag = bv.get1Bit()
	}

	return spsData
}

func (p *H264VideoStreamParser) afterGetting()      {}
func (p *H264VideoStreamParser) doGetNextFrame()    {}
func (p *H264VideoStreamParser) stopGettingFrames() {}
func (p *H264VideoStreamParser) maxFrameSize() uint { return 0 }
func (p *H264VideoStreamParser) GetNextFrame(buffTo []byte, maxSize uint,
	afterGettingFunc, onCloseFunc interface{}) {
}

//////// H264VideoStreamFramer ////////
type H264VideoStreamFramer struct {
	MPEGVideoStreamFramer
	frameRate            float64
	lastSeenSPS          []byte
	lastSeenPPS          []byte
	lastSeenSPSSize      uint
	lastSeenPPSSize      uint
	nextPresentationTime sys.Timeval
}

func newH264VideoStreamFramer(inputSource IFramedSource) *H264VideoStreamFramer {
	framer := new(H264VideoStreamFramer)
	framer.inputSource = inputSource
	framer.frameRate = 25.0
	framer.initMPEGVideoStreamFramer(newH264VideoStreamParser(inputSource))
	framer.InitFramedSource(framer)
	return framer
}

func (f *H264VideoStreamFramer) getSPSandPPS(sps, pps string, spsSize, ppsSize uint) {
	sps = string(f.lastSeenSPS)
	pps = string(f.lastSeenPPS)
	spsSize = f.lastSeenSPSSize
	ppsSize = f.lastSeenPPSSize
}

func (f *H264VideoStreamFramer) setSPSandPPS(sPropParameterSetsStr string) {
	sPropRecords, numSPropRecords := parseSPropParameterSets(sPropParameterSetsStr)
	var i uint
	for i = 0; i < numSPropRecords; i++ {
		if sPropRecords[i].sPropLength == 0 {
			continue
		}

		nalUnitType := (sPropRecords[i].sPropBytes[0]) & 0x1F
		if nalUnitType == 7 { /* SPS */
			f.saveCopyOfSPS(sPropRecords[i].sPropBytes, sPropRecords[i].sPropLength)
		} else if nalUnitType == 8 { /* PPS */
			f.saveCopyOfPPS(sPropRecords[i].sPropBytes, sPropRecords[i].sPropLength)
		}
	}
}

func (f *H264VideoStreamFramer) saveCopyOfSPS(from []byte, size uint) {
	f.lastSeenSPS = make([]byte, size)
	f.lastSeenSPS = from
	f.lastSeenSPSSize = size
}

func (f *H264VideoStreamFramer) saveCopyOfPPS(from []byte, size uint) {
	f.lastSeenPPS = make([]byte, size)
	f.lastSeenPPS = from
	f.lastSeenPPSSize = size
}

func (f *H264VideoStreamFramer) setPresentationTime() {
	f.presentationTime = f.nextPresentationTime
}
