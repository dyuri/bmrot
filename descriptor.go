// Copyright 2020 Frederik Zipp. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
)

// A Descriptor holds metadata for a bitmap font.
type Descriptor struct {
	Info    Info
	Common  Common
	Pages   map[int]Page
	Chars   map[rune]Char
	Kerning map[CharPair]Kerning
}

func (d *Descriptor) String() string {
	pages := ""
	for _, page := range d.Pages {
		pages += fmt.Sprintf("%s\n", page.String())
	}
	chars := fmt.Sprintf("chars count=%d\n", len(d.Chars))
	for _, char := range d.Chars {
		chars += fmt.Sprintf("%s\n", char.String())
	}
	return fmt.Sprintf("%s\n%s\n%s%s", d.Info.String(), printCommon(&d.Common, &d.Pages), pages, chars)
}

func printCommon(c *Common, p *map[int]Page) string {
	return fmt.Sprintf("common lineHeight=%d base=%d scaleW=%d scaleH=%d pages=%d packed=%d alphaChnl=%d redChnl=%d greenChnl=%d blueChnl=%d",
		c.LineHeight,
		c.Base,
		c.ScaleW,
		c.ScaleH,
		len(*p),
		boolToInt(c.Packed),
		c.AlphaChannel,
		c.RedChannel,
		c.GreenChannel,
		c.BlueChannel,
	)
}

// Rotate rotates the font (descriptor) 90 degrees clockwise.
func (d *Descriptor) Rotate() {
	d.Info.Padding = Padding{d.Info.Padding.Left, d.Info.Padding.Up, d.Info.Padding.Right, d.Info.Padding.Down}
	d.Info.Spacing = Spacing{d.Info.Spacing.Vertical, d.Info.Spacing.Horizontal}
	d.Common.ScaleW, d.Common.ScaleH = d.Common.ScaleH, d.Common.ScaleW
	lh := 0
	for _, char := range d.Chars {
		char.X, char.Y = d.Common.ScaleW-char.Y-char.Height, char.X
		char.XOffset, char.YOffset = char.YOffset, char.XOffset
		char.Width, char.Height = char.Height, char.Width
		char.XAdvance = char.Width + char.XOffset
		if lh < char.Height {
			lh = char.Height
		}
		d.Chars[char.ID] = char
	}
	d.Common.LineHeight = lh
	d.Common.Base = lh
}

type Info struct {
	Face     string
	Size     int
	Bold     bool
	Italic   bool
	Charset  string
	Unicode  bool
	StretchH int
	Smooth   bool
	AA       int
	Padding  Padding
	Spacing  Spacing
	Outline  int
}

type Padding struct {
	Up, Right, Down, Left int
}

type Spacing struct {
	Horizontal, Vertical int
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (i *Info) String() string {
	return fmt.Sprintf("info face=\"%s\" size=%d bold=%d italic=%d charset=\"%s\" unicode=%d stretchH=%d smooth=%d aa=%d padding=%d,%d,%d,%d spacing=%d,%d outline=%d",
		i.Face,
		i.Size,
		boolToInt(i.Bold),
		boolToInt(i.Italic),
		i.Charset,
		boolToInt(i.Unicode),
		i.StretchH,
		boolToInt(i.Smooth),
		i.AA,
		i.Padding.Up,
		i.Padding.Right,
		i.Padding.Down,
		i.Padding.Left,
		i.Spacing.Horizontal,
		i.Spacing.Vertical,
		i.Outline,
	)
}

type Common struct {
	LineHeight   int
	Base         int
	ScaleW       int
	ScaleH       int
	Packed       bool
	AlphaChannel ChannelInfo
	RedChannel   ChannelInfo
	GreenChannel ChannelInfo
	BlueChannel  ChannelInfo
}

func (c *Common) Scale() image.Point {
	return image.Pt(c.ScaleH, c.ScaleH)
}

type ChannelInfo int

const (
	Glyph ChannelInfo = iota
	Outline
	GlyphAndOutline
	Zero
	One
)

type Page struct {
	ID   int
	File string
}

func (p *Page) String() string {
	return fmt.Sprintf("page id=%d file=\"%s\"", p.ID, p.File)
}

type Char struct {
	ID       rune
	X        int
	Y        int
	Width    int
	Height   int
	XOffset  int
	YOffset  int
	XAdvance int
	Page     int
	Channel  Channel
}

func (c *Char) String() string {
	return fmt.Sprintf("char id=%d x=%d y=%d width=%d height=%d xoffset=%d yoffset=%d xadvance=%d page=%d chnl=%d",
		c.ID,
		c.X,
		c.Y,
		c.Width,
		c.Height,
		c.XOffset,
		c.YOffset,
		c.XAdvance,
		c.Page,
		c.Channel,
	)
}

func (c *Char) Pos() image.Point {
	return image.Pt(c.X, c.Y)
}

func (c *Char) Size() image.Point {
	return image.Pt(c.Width, c.Height)
}

func (c *Char) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: c.Pos(),
		Max: c.Pos().Add(c.Size()),
	}
}

func (c *Char) Offset() image.Point {
	return image.Pt(c.XOffset, c.YOffset)
}

type Channel int

const (
	Blue  Channel = 1
	Green Channel = 2
	Red   Channel = 4
	Alpha Channel = 8
	All   Channel = 15
)

// CharPair is a pair of characters. It is used as the key in the font's
// kerning map.
type CharPair struct {
	First, Second rune
}

// Kerning is a horizontal offset in pixels to be used if a specific character
// pair occurs when drawing text. It is used for the values in the font's
// kerning map.
type Kerning struct {
	Amount int
}

func closeChecked(c io.Closer, err *error) {
	cErr := c.Close()
	if cErr != nil && *err == nil {
		*err = cErr
	}
}

// LoadDescriptor loads the font descriptor data from a BMFont descriptor file in
// text format (usually with the file extension .fnt). It does not load the
// referenced page sheet images. If you also want to load the page sheet
// images, use the Load function to get a complete BitmapFont instance.
func LoadDescriptor(path string) (d *Descriptor, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer closeChecked(f, &err)
	return parseDescriptor(filepath.Base(path), f)
}

// ReadDescriptor parses font descriptor data in BMFont's text format from a
// reader. It does not load the referenced page sheet images. If you also want
// to load the page sheet images, use the Load function to get a complete
// BitmapFont instance.
func ReadDescriptor(r io.Reader) (d *Descriptor, err error) {
	return parseDescriptor("bmfont", r)
}
