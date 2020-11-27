// https://github.com/f-secure-foundry/armory-boot
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//+build armory

package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/f-secure-foundry/tamago/soc/imx6"
	"github.com/f-secure-foundry/tamago/soc/imx6/usdhc"

	usbarmory "github.com/f-secure-foundry/tamago/board/f-secure/usbarmory/mark-two"
)

var Build string
var Revision string

var Boot string
var StartKernel string
var LenKernel string
var StartProof string

func init() {
	log.SetFlags(0)

	if err := imx6.SetARMFreq(900); err != nil {
		panic(fmt.Sprintf("WARNING: error setting ARM frequency: %v\n", err))
	}
}

func main() {
	usbarmory.LED("blue", false)
	usbarmory.LED("white", false)

	var card *usdhc.USDHC

	switch Boot {
	case "eMMC":
		card = usbarmory.MMC
	case "uSD":
		card = usbarmory.SD
	default:
		panic("invalid boot parameter")
	}

	if err := card.Detect(); err != nil {
		panic(fmt.Sprintf("card detect error: %v\n", err))
	}

	kernelOffset, err := strconv.ParseInt(StartKernel, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid kernel partition start offset: %v\n", err))
	}
	kernelLen, err := strconv.ParseInt(LenKernel, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid kernel partition length : %v\n", err))
	}
	proofOffset, err := strconv.ParseInt(StartProof, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid proof partition start offset: %v\n", err))
	}

	kernelPart := &Partition{
		Card:   card,
		Offset: kernelOffset,
	}
	if err = conf.Read(kernelPart, defaultConfigPath); err != nil {
		panic(fmt.Sprintf("invalid configuration: %v\n", err))
	}

	usbarmory.LED("white", true)
	h, err := hashPartition(kernelLen, kernelPart)
	if err != nil {
		panic(fmt.Sprintf("failed to hash kernelPart: %w\n", err))
	}
	fmt.Printf("Partition hash: %x\n", h)
	usbarmory.LED("white", false)

	proofPart := &Partition{
		Card:   card,
		Offset: proofOffset,
	}
	bundle, err := loadBundle(proofPart)
	if err != nil {
		panic(fmt.Sprintf("Failed to load proof bundle: %q", err))
	}
	fmt.Printf("Loaded bundle: %v\n", bundle)



	if len(PublicKeyStr) > 0 {
		valid, err := conf.Verify(kernelPart, defaultConfigPath+signatureSuffix)

		if err != nil {
			panic(fmt.Sprintf("configuration verification error: %v\n", err))
		}

		if !valid {
			panic("invalid configuration signature")
		}
	}

	kernel, err := kernelPart.ReadAll(conf.Kernel[0])

	if err != nil {
		panic(fmt.Sprintf("invalid kernel path: %v\n", err))
	}

	dtb, err := kernelPart.ReadAll(conf.DeviceTreeBlob[0])

	if err != nil {
		panic(fmt.Sprintf("invalid dtb path: %v\n", err))
	}

	usbarmory.LED("blue", true)

	if !verifyHash(kernel, conf.Kernel[1]) {
		panic("invalid kernel hash")
	}

	if !verifyHash(dtb, conf.DeviceTreeBlob[1]) {
		panic("invalid dtb hash")
	}

	dtb, err = fixupDeviceTree(dtb, conf.CmdLine)

	if err != nil {
		panic(fmt.Sprintf("dtb fixup error: %v\n", err))
	}

	boot(kernel, dtb, conf.CmdLine)
}
