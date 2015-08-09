package liveMedia

import (
	"fmt"
	. "include"
)

//////// H264VideoStreamParser ////////
type H264VideoStreamParser struct {
	MPEGVideoStreamParser
	outputStartCodeSize        int
	haveSeenFirstStartCode     bool
	haveSeenFirstByteOfNALUnit bool
}

func NewH264VideoStreamParser() *H264VideoStreamParser {
	return new(H264VideoStreamParser)
}

func (this *H264VideoStreamParser) parse() uint {
	if !this.haveSeenFirstStartCode {
		for first4Bytes := this.test4Bytes(); first4Bytes != 0x00000001; {
			this.get1Byte()
			this.setParseState()
			fmt.Println("parse", first4Bytes)
		}

		this.skipBytes(4)
		this.haveSeenFirstStartCode = true
	}

	if this.outputStartCodeSize > 0 && this.curFrameSize() == 0 && !this.HaveSeenEOF() {
		// Include a start code in the output:
		this.save4Bytes(0x00000001)
	}
	/*
	       if (haveSeenEOF()) {
	         // We hit EOF the last time that we tried to parse this data, so we know that any remaining unparsed data
	         // forms a complete NAL unit, and that there's no 'start code' at the end:
	         remainingDataSize := totNumValidBytes() - curOffset()
	         for ; remainingDataSize > 0; {
	             nextByte = get1Byte()
	             if !this.haveSeenFirstByteOfNALUnit {
	                 this.firstByteOfNALUnit = nextByte
	                 this.haveSeenFirstByteOfNALUnit = true
	             }
	             saveByte(nextByte)
	             remainingDataSize--
	         }

	         get1Byte(); // forces another read, which will cause EOF to get handled for real this time
	         return
	       } else {
	           next4Bytes = test4Bytes()
	           if !this.haveSeenFirstByteOfNALUnit {
	               this.firstByteOfNALUnit = next4Bytes>>24
	               this.haveSeenFirstByteOfNALUnit = true
	           }
	           for next4Bytes != 0x00000001 && (next4Bytes&0xFFFFFF00) != 0x00000100 {
	               // We save at least some of "next4Bytes".
	               if next4Bytes&0xFF > 1 {
	                   // Common case: 0x00000001 or 0x000001 definitely doesn't begin anywhere in "next4Bytes", so we save all of it:
	                   save4Bytes(next4Bytes)
	                   skipBytes(4)
	               } else {
	                   // Save the first byte, and continue testing the rest:
	                   saveByte(next4Bytes>>24)
	                   skipBytes(1)
	               }
	               setParseState() // ensures forward progress
	               next4Bytes = test4Bytes()
	           }
	           // Assert: next4Bytes starts with 0x00000001 or 0x000001, and we've saved all previous bytes (forming a complete NAL unit).
	           // Skip over these remaining bytes, up until the start of the next NAL unit:
	           if next4Bytes == 0x00000001 {
	   	        skipBytes(4)
	           } else {
	               skipBytes(3)
	           }
	       }

	       nal_ref_idc = this.firstByteOfNALUnit&0x60>>5
	       nal_unit_type = this.firstByteOfNALUnit&0x1F
	       this.haveSeenFirstByteOfNALUnit = false // for the next NAL unit that we parse

	       switch nal_unit_type {
	           case 6:     // Supplemental enhancement information (SEI)
	               analyze_sei_data()
	               // Later, perhaps adjust "fPresentationTime" if we saw a "pic_timing" SEI payload??? #####
	           case 7:     // Sequence parameter set
	               // First, save a copy of this NAL unit, in case the downstream object wants to see it:
	   	        usingSource().saveCopyOfSPS(fStartOfFrame + fOutputStartCodeSize, fTo - fStartOfFrame - fOutputStartCodeSize);

	               // Parse this NAL unit to check whether frame rate information is present:
	               //num_units_in_tick, time_scale, fixed_frame_rate_flag
	               //analyze_seq_parameter_set_data(num_units_in_tick, time_scale, fixed_frame_rate_flag)
	               if (time_scale > 0 && num_units_in_tick > 0) {
	                   //usingSource().frameRate = time_scale/(2.0*num_units_in_tick)
	   	        } else {
	   	        }
	           case 8:// Picture parameter set
	               // Save a copy of this NAL unit, in case the downstream object wants to see it:
	               //usingSource()->saveCopyOfPPS(fStartOfFrame + fOutputStartCodeSize, fTo - fStartOfFrame - fOutputStartCodeSize)
	       }

	       //usingSource()->setPresentationTime()

	       thisNALUnitEndsAccessUnit := false; // until we learn otherwise
	       if haveSeenEOF() {
	         // There is no next NAL unit, so we assume that this one ends the current 'access unit':
	         thisNALUnitEndsAccessUnit = True;
	       } else {
	           isVCL := nal_unit_type <= 5 && nal_unit_type > 0    // Would need to include type 20 for SVC and MVC #####
	           if isVCL {
	               testBytes(&firstByteOfNextNALUnit, 1)
	               next_nal_ref_idc = (firstByteOfNextNALUnit&0x60)>>5
	               next_nal_unit_type = firstByteOfNextNALUnit&0x1F
	               if (next_nal_unit_type >= 6) {
	                   // The next NAL unit is not a VCL; therefore, we assume that this NAL unit ends the current 'access unit':
	                   thisNALUnitEndsAccessUnit = true
	               } else {
	                   // The next NAL unit is also a VCL.  We need to examine it a little to figure out if it's a different 'access unit'.
	                   // (We use many of the criteria described in section 7.4.1.2.4 of the H.264 specification.)
	                   IdrPicFlag = nal_unit_type == 5
	                   next_IdrPicFlag = next_nal_unit_type == 5
	                   if (next_IdrPicFlag != IdrPicFlag) {
	                       // IdrPicFlag differs in value
	   	                thisNALUnitEndsAccessUnit = true
	   	            } else if (next_nal_ref_idc != nal_ref_idc && next_nal_ref_idc*nal_ref_idc == 0) {
	   	                // nal_ref_idc differs in value with one of the nal_ref_idc values being equal to 0
	   	                thisNALUnitEndsAccessUnit = true
	   	            } else if (nal_unit_type == 1 ||
	                              nal_unit_type == 2 ||
	                              nal_unit_type == 5) && (next_nal_unit_type == 1 ||
	                                                      next_nal_unit_type == 2 ||
	                                                      next_nal_unit_type == 5) {
	   	                // Both this and the next NAL units begin with a "slice_header".
	   	                // Parse this (for each), to get parameters that we can compare:

	   	                // Current NAL unit's "slice_header":
	   	                analyze_slice_header(fStartOfFrame + fOutputStartCodeSize, fTo, nal_unit_type, frame_num, pic_parameter_set_id, idr_pic_id, field_pic_flag, bottom_field_flag)

	   	                // Next NAL unit's "slice_header":
	                       testBytes(next_slice_header)
	                       //analyze_slice_header(next_slice_header, &next_slice_header[sizeof next_slice_header], next_nal_unit_type, next_frame_num, next_pic_parameter_set_id, next_idr_pic_id, next_field_pic_flag, next_bottom_field_flag)

	                       if next_frame_num != frame_num {
	                           // frame_num differs in value
	                           thisNALUnitEndsAccessUnit = true
	                       } else if next_pic_parameter_set_id != pic_parameter_set_id {
	                           // pic_parameter_set_id differs in value
	                           thisNALUnitEndsAccessUnit = true
	                       } else if next_field_pic_flag != field_pic_flag {
	                           // field_pic_flag differs in value
	                           thisNALUnitEndsAccessUnit = true
	                       } else if (next_bottom_field_flag != bottom_field_flag) {
	                           // bottom_field_flag differs in value
	                           thisNALUnitEndsAccessUnit = true
	                       } else if next_IdrPicFlag == 1 && next_idr_pic_id != idr_pic_id {
	                           // IdrPicFlag is equal to 1 for both and idr_pic_id differs in value
	                           // Note: We already know that IdrPicFlag is the same for both.
	                           thisNALUnitEndsAccessUnit = true
	                       }
	   	            }
	   	        }
	           }
	       }

	       if thisNALUnitEndsAccessUnit {
	         usingSource().fPictureEndMarker = true
	         usingSource().fPictureCount++

	         // Note that the presentation time for the next NAL unit will be different:
	         nextPT = usingSource().fNextPresentationTime // alias
	         nextPT = usingSource().fPresentationTime
	         nextFraction = nextPT.tv_usec/1000000.0 + 1/usingSource().fFrameRate
	         nextSecsIncrement = nextFraction
	         nextPT.tv_sec += nextSecsIncrement
	         nextPT.tv_usec = (nextFraction - nextSecsIncrement)*1000000
	       }
	       setParseState()

	       return curFrameSize()*/
	return 0
}

func (this *H264VideoStreamParser) analyzeSPSData() {
}

//////// H264VideoStreamFramer ////////
type H264VideoStreamFramer struct {
	MPEGVideoStreamFramer
	parser               *H264VideoStreamParser
	nextPresentationTime Timeval
	lastSeenSPS          []byte
	lastSeenPPS          []byte
	lastSeenSPSSize      uint
	lastSeenPPSSize      uint
	frameRate            float64
}

func NewH264VideoStreamFramer(inputSource IFramedSource) *H264VideoStreamFramer {
	h264VideoStreamFramer := new(H264VideoStreamFramer)
	h264VideoStreamFramer.parser = NewH264VideoStreamParser()
	h264VideoStreamFramer.inputSource = inputSource
	h264VideoStreamFramer.frameRate = 25.0
	h264VideoStreamFramer.InitMPEGVideoStreamFramer(h264VideoStreamFramer.parser)
	return h264VideoStreamFramer
}

func (this *H264VideoStreamFramer) getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}) {
	fmt.Println("H264VideoStreamFramer::getNextFrame")
	this.inputSource.getNextFrame(buffTo, maxSize, afterGettingFunc)
}

func (this *H264VideoStreamFramer) setSPSandPPS(sPropParameterSetsStr string) {
	sPropRecords, numSPropRecords := parseSPropParameterSets(sPropParameterSetsStr)
	var i uint
	for i = 0; i < numSPropRecords; i++ {
		if sPropRecords[i].sPropLength == 0 {
			continue
		}

		nalUnitType := (sPropRecords[i].sPropBytes[0]) & 0x1F
		if nalUnitType == 7 { /* SPS */
			this.saveCopyOfSPS(sPropRecords[i].sPropBytes, sPropRecords[i].sPropLength)
		} else if nalUnitType == 8 { /* PPS */
			this.saveCopyOfPPS(sPropRecords[i].sPropBytes, sPropRecords[i].sPropLength)
		}
	}
}

func (this *H264VideoStreamFramer) saveCopyOfSPS(from []byte, size uint) {
	this.lastSeenSPS = make([]byte, size)
	this.lastSeenSPS = from
	this.lastSeenSPSSize = size
}

func (this *H264VideoStreamFramer) saveCopyOfPPS(from []byte, size uint) {
	this.lastSeenPPS = make([]byte, size)
	this.lastSeenPPS = from
	this.lastSeenPPSSize = size
}
