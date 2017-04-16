package livemedia

import (
	"errors"
	sys "syscall"

	"github.com/djwackey/gitea/log"
)

const (
	spsMaxSize uint = 1000
	seiMaxSize uint = 5000 // larger than the largest possible SEI NAL unit

	numNextSliceHeaderBytesToAnalyze uint = 12
)

//////// H264VideoStreamParser ////////
type H264VideoStreamParser struct {
	MPEGVideoStreamParser
	outputStartCodeSize        uint
	firstByteOfNALUnit         uint
	log2MaxFrameNum            uint
	frameMbsOnlyFlag           bool
	separateColourPlaneFlag    bool
	includeStartCodeInOutput   bool
	haveSeenFirstStartCode     bool
	haveSeenFirstByteOfNALUnit bool
}

func newH264VideoStreamParser(usingSource, inputSource IFramedSource,
	clientOnInputCloseFunc interface{}) *H264VideoStreamParser {
	parser := new(H264VideoStreamParser)
	parser.log2MaxFrameNum = 5
	parser.frameMbsOnlyFlag = true
	parser.initMPEGVideoStreamParser(usingSource, inputSource, clientOnInputCloseFunc)
	if parser.includeStartCodeInOutput {
		parser.outputStartCodeSize = 4
	}
	return parser
}

func (p *H264VideoStreamParser) UsingSource() *H264VideoStreamFramer {
	return p.usingSource.(*H264VideoStreamFramer)
}

func (p *H264VideoStreamParser) parse() (uint, error) {
	log.Trace("[H264VideoStreamParser] is's parsing h264 file video stream.")

	var err error
	// The stream must start with a 0x00000001
	if !p.haveSeenFirstStartCode {
		var first4Bytes uint
		for {
			if first4Bytes, err = p.test4Bytes(); err != nil {
				log.Debug("failed to test of start of stream: %s", err.Error())
				return 0, err
			}

			if first4Bytes == 0x00000001 {
				break
			}

			if _, err = p.get1Byte(); err != nil {
				log.Debug("failed to check first 4 bytes: %s", err.Error())
				return 0, err
			}

			p.setParseState()
		}

		// skip this initial code
		if err = p.skipBytes(4); err != nil {
			log.Error(0, "skip this initial code: %s", err.Error())
			return 0, err
		}
		p.haveSeenFirstStartCode = true
	}

	if p.outputStartCodeSize > 0 && p.curFrameSize() == 0 && !p.haveSeenEOF {
		// Include a start code in the output:
		p.save4Bytes(0x00000001)
	}

	var nextByte, next4Bytes, skipBytes uint
	if p.haveSeenEOF {
		// We hit EOF the last time that we tried to parse this data, so we know that any remaining unparsed data
		// forms a complete NAL unit, and that there's no 'start code' at the end:
		remainingDataSize := p.totNumValidBytes - p.curOffset()
		for remainingDataSize > 0 {
			if nextByte, err = p.get1Byte(); err != nil {
				log.Fatal(0, "failed to get 1 byte from remaining data: %s", err.Error())
				return 0, err
			}

			if !p.haveSeenFirstByteOfNALUnit {
				p.firstByteOfNALUnit = nextByte
				p.haveSeenFirstByteOfNALUnit = true
			}
			p.saveByte(nextByte)
			remainingDataSize--
		}

		p.get1Byte() // forces another read, which will cause EOF to get handled for real this time
		return 0, errors.New("EOF")
	} else {
		if next4Bytes, err = p.test4Bytes(); err != nil {
			log.Error(0, "failed to test next 4 bytes: %s", err.Error())
			return 0, err
		}
		//log.Debug("progress next 4 bytes: %d", next4Bytes)

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
				skipBytes = 4
			} else {
				// Save the first byte, and continue testing the rest:
				p.saveByte(next4Bytes >> 24)
				skipBytes = 1
			}

			if err = p.skipBytes(skipBytes); err != nil {
				log.Warn("failed to skip 1 or 4 bytes: %s", err.Error())
				return 0, err
			}

			p.setParseState() // ensures forward progress
			if next4Bytes, err = p.test4Bytes(); err != nil {
				log.Warn("failed to test 4 bytes(), ensures forward progress: %s", err.Error())
				return 0, err
			}
		}

		// Assert: next4Bytes starts with 0x00000001 or 0x000001,
		// and we've saved all previous bytes (forming a complete NAL unit).
		// Skip over these remaining bytes, up until the start of the next NAL unit:
		if next4Bytes == 0x00000001 {
			skipBytes = 4
		} else {
			skipBytes = 3
		}

		if err = p.skipBytes(skipBytes); err != nil {
			log.Error(0, "failed to skip 3 or 4 bytes: %s", err.Error())
			return 0, err
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
		size := p.numSavedBytes - p.outputStartCodeSize
		p.UsingSource().saveCopyOfSPS(p.startOfFrame[p.outputStartCodeSize:], size)

		// Parse this NAL unit to check whether frame rate information is present:
		spsData := p.analyzeSPSData()
		if spsData.timeScale > 0 && spsData.numUnitsInTick > 0 {
			p.UsingSource().frameRate = spsData.timeScale / (2.0 * spsData.numUnitsInTick)
			log.Debug("Get the frameRate(%d) from Sequence parameter set", p.UsingSource().frameRate)
		} else {
		}
	case 8: // Picture parameter set
		// Save a copy of this NAL unit, in case the downstream object wants to see it:
		size := p.numSavedBytes - p.outputStartCodeSize
		p.UsingSource().saveCopyOfPPS(p.startOfFrame[p.outputStartCodeSize:], size)
	}

	p.UsingSource().setPresentationTime()
	log.Trace("\tPresentation Time: %d.%06d", p.UsingSource().presentationTime.Sec,
		p.UsingSource().presentationTime.Usec)

	thisNALUnitEndsAccessUnit := false // until we learn otherwise
	if p.haveSeenEOF {
		// There is no next NAL unit, so we assume that this one ends the current 'access unit':
		thisNALUnitEndsAccessUnit = true
	} else {
		isVCL := nalUnitType <= 5 && nalUnitType > 0 // Would need to include type 20 for SVC and MVC #####
		if isVCL {
			firstByteOfNextNALUnit := make([]byte, 1)
			if err = p.testBytes(firstByteOfNextNALUnit, 1); err != nil {
				log.Error(0, "failed to test bytes(firstByteOfNextNALUnit): %s", err.Error())
				return 0, err
			}

			nextNalRefIdc := uint((firstByteOfNextNALUnit[0] & 0x60) >> 5)
			nextNalUnitType := uint(firstByteOfNextNALUnit[0] & 0x1F)
			if nextNalUnitType >= 6 {
				// The next NAL unit is not a VCL;
				// therefore, we assume that this NAL unit ends the current 'access unit':
				thisNALUnitEndsAccessUnit = true
			} else {
				// The next NAL unit is also a VCL.
				// We need to examine it a little to figure out if it's a different 'access unit'.
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
					nalUnitType == 5) &&
					(nextNalUnitType == 1 ||
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
		p.UsingSource().pictureEndMarker = true
		p.UsingSource().pictureCount++

		// Note that the presentation time for the next NAL unit will be different:
		p.UsingSource().nextPresentationTime = p.UsingSource().presentationTime

		nextFraction := float32(p.UsingSource().nextPresentationTime.Usec)/1000000.0 + 1/float32(p.UsingSource().frameRate)
		nextSecsIncrement := float32(uint(nextFraction))
		p.UsingSource().nextPresentationTime.Sec += int64(nextSecsIncrement)
		p.UsingSource().nextPresentationTime.Usec = int64((nextFraction - nextSecsIncrement) * 1000000)
	}
	p.setParseState()
	return p.curFrameSize(), nil
}

func (p *H264VideoStreamParser) removeEmulationBytes(nalUnitCopy []byte, maxSize uint) uint {
	nalUnitOrig := p.startOfFrame[p.outputStartCodeSize:]

	numBytesInNALUnit := p.numSavedBytes
	if numBytesInNALUnit > maxSize {
		return 0
	}
	var nalUnitCopySize, i uint
	for i = 0; i < numBytesInNALUnit; i++ {
		if i+2 < numBytesInNALUnit && nalUnitOrig[i] == 0 && nalUnitOrig[i+1] == 0 && nalUnitOrig[i+2] == 3 {
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
	totNumBits := 8 * (p.numSavedBytes - p.outputStartCodeSize)
	bv := newBitVector(start, 0, totNumBits)

	// Some of the result parameters might not be present in the header; set them to default values:
	header := new(sliceHeader)

	// Note: We assume that there aren't any 'emulation prevention' bytes here to worry about...
	bv.skipBits(8) // forbidden_zero_bit; nal_ref_idc; nal_unit_type
	firstMBInSlice := bv.getExpGolomb()
	sliceType := bv.getExpGolomb()
	log.Trace("firstMBInSlice: %d, sliceType: %d", firstMBInSlice, sliceType)
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
	sps := make([]byte, spsMaxSize)
	spsSize := p.removeEmulationBytes(sps, spsMaxSize)

	bv := newBitVector(sps, 0, 8*spsSize)

	bv.skipBits(8) // forbidden_zero_bit; nal_ref_idc; nal_unit_type
	profileIdc := bv.getBits(8)
	log.Trace("profileIdc:%d", profileIdc)
	constraintSetNFlag := bv.getBits(8) // also "reserved_zero_2bits" at end
	log.Trace("constraintSetNFlag:%d", constraintSetNFlag)
	levelIdc := bv.getBits(8)
	log.Trace("levelIdc:%d", levelIdc)
	seqParameterSetID := bv.getExpGolomb()
	log.Trace("seqParameterSetID:%d", seqParameterSetID)
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
			log.Trace("eparateColourPlaneFlag:%d", eparateColourPlaneFlag)
		}

		bv.getExpGolomb() // bit_depth_luma_minus8
		bv.getExpGolomb() // bit_depth_chroma_minus8
		bv.skipBits(1)    // qpprime_y_zero_transform_bypass_flag

		seqScalingMatrixPresentFlag := bv.get1Bit()
		log.Trace("seqScalingMatrixPresentFlag:%d", seqScalingMatrixPresentFlag)
		if seqScalingMatrixPresentFlag != 0 {
			cond := 12
			if chromaFormatIdc != 3 {
				cond = 8
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
	}

	log2MaxFrameNumMinus4 := bv.getExpGolomb()
	log.Trace("log2MaxFrameNumMinus4:%d", log2MaxFrameNumMinus4)
	p.log2MaxFrameNum = log2MaxFrameNumMinus4 + 4
	picOrderCntType := bv.getExpGolomb()
	log.Trace("picOrderCntType:%d", picOrderCntType)
	if picOrderCntType == 0 {
		log2MaxPicOrderCntLsbMinus4 := bv.getExpGolomb()
		log.Trace("log2MaxPicOrderCntLsbMinus4:%d", log2MaxPicOrderCntLsbMinus4)
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
	log.Trace("maxNumRefFrames:%d", maxNumRefFrames)
	gapsInFrameNumValueAllowedFlag := bv.get1Bit()
	log.Trace("gapsInFrameNumValueAllowedFlag:%d", gapsInFrameNumValueAllowedFlag)
	picWidthInMbsMinus1 := bv.getExpGolomb()
	log.Trace("picWidthInMbsMinus1:%d", picWidthInMbsMinus1)
	picHeightInMapUnitsMinus1 := bv.getExpGolomb()
	log.Trace("picHeightInMapUnitsMinus1:%d", picHeightInMapUnitsMinus1)
	frameMbsOnlyFlag := bv.get1BitBoolean()
	log.Trace("frameMbsOnlyFlag:%d", frameMbsOnlyFlag)
	if !frameMbsOnlyFlag {
		bv.skipBits(1) // mb_adaptive_frame_field_flag
	}
	bv.skipBits(1) // direct_8x8_inference_flag
	frameCroppingFlag := bv.get1Bit()
	log.Trace("frameCroppingFlag:%d", frameCroppingFlag)
	if frameCroppingFlag != 0 {
		bv.getExpGolomb() // frame_crop_left_offset
		bv.getExpGolomb() // frame_crop_right_offset
		bv.getExpGolomb() // frame_crop_top_offset
		bv.getExpGolomb() // frame_crop_bottom_offset
	}

	vuiParametersPresentFlag := bv.get1Bit()
	log.Trace("vuiParametersPresentFlag:%d", vuiParametersPresentFlag)

	var spsData *seqParameterSet
	if vuiParametersPresentFlag != 0 {
		spsData = p.analyzeVUIParameters(bv)
	} else {
		spsData = new(seqParameterSet)
	}

	return spsData
}

func (p *H264VideoStreamParser) analyzeSEIData() {
	// Begin by making a copy of the NAL unit data, removing any 'emulation prevention' bytes:
	sei := make([]byte, seiMaxSize)
	seiSize := p.removeEmulationBytes(sei, seiMaxSize)

	// skip the initial byte (forbidden_zero_bit; nal_ref_idc; nal_unit_type); we've already seen it
	var j uint = 1
	for j < seiSize {
		var payloadSize uint
		//var payloadType uint
		for {
			//payloadType += uint(sei[j])
			if sei[j] == 255 && j < seiSize {
				j++
			} else {
				j++
				break
			}
		}

		if j >= seiSize {
			break
		}

		for {
			payloadSize += uint(sei[j])
			if sei[j] == 255 && j < seiSize {
				j++
			} else {
				j++
				break
			}
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
	chromaLocInfoPresentFlag := bv.get1Bit()
	if chromaLocInfoPresentFlag != 0 {
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

//////// H264VideoStreamFramer ////////
type H264VideoStreamFramer struct {
	MPEGVideoStreamFramer
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
	framer.initMPEGVideoStreamFramer(newH264VideoStreamParser(framer, inputSource, framer.handleClosure))
	framer.initFramedSource(framer)
	framer.nextPresentationTime = framer.presentationTimeBase
	return framer
}

func (f *H264VideoStreamFramer) destroy() {
	f.inputSource.destroy()
	f.stopGettingFrames()
}

func (f *H264VideoStreamFramer) getSPSandPPS() (sps, pps []byte, spsSize, ppsSize uint) {
	sps, pps = f.lastSeenSPS, f.lastSeenPPS
	spsSize, ppsSize = f.lastSeenSPSSize, f.lastSeenPPSSize
	return
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
	f.lastSeenSPSSize = size
	copy(f.lastSeenSPS, from)
}

func (f *H264VideoStreamFramer) saveCopyOfPPS(from []byte, size uint) {
	f.lastSeenPPS = make([]byte, size)
	f.lastSeenPPSSize = size
	copy(f.lastSeenPPS, from)
}

func (f *H264VideoStreamFramer) setPresentationTime() {
	f.presentationTime = f.nextPresentationTime
}
