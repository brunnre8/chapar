package pages

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"net/http"
	"time"

	"github.com/mirzakhany/chapar/internal/notify"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/richtext"
	"github.com/dustin/go-humanize"
	"github.com/mirzakhany/chapar/internal/rest"
	"github.com/mirzakhany/chapar/ui/widgets"
	"golang.design/x/clipboard"
)

var (
	requestTabsInset = layout.Inset{Left: unit.Dp(5), Top: unit.Dp(3)}
)

type RestContainer struct {
	methodDropDown *widgets.DropDown
	address        *widget.Editor

	copyResponseButton *widgets.FlatButton
	saveResponseButton *widgets.FlatButton

	section      *widget.Editor
	requestBody  *widget.Editor
	responseBody *widget.Editor
	list         *widget.List

	responseMacro op.CallOp
	responseDim   layout.Dimensions

	richResponse richtext.InteractiveText
	spans        []richtext.SpanStyle

	sendClickable widget.Clickable
	sendButton    material.ButtonStyle

	loading       bool
	resultUpdated bool
	result        string

	resultStatus string

	split widgets.SplitView

	requestTabs  *widgets.Tabs
	responseTabs *widgets.Tabs

	preRequestDropDown *widgets.DropDown
	preRequestBody     *widget.Editor

	postRequestDropDown *widgets.DropDown
	postRequestBody     *widget.Editor

	requestBodyDropDown *widgets.DropDown

	notification *widgets.Notification

	params *widgets.KeyValue
}

func NewRestContainer(theme *material.Theme) *RestContainer {
	r := &RestContainer{
		split: widgets.SplitView{
			Ratio:         0,
			BarWidth:      unit.Dp(2),
			BarColor:      color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			BarColorHover: theme.Palette.ContrastBg,
		},
		responseBody:    new(widget.Editor),
		address:         new(widget.Editor),
		requestBody:     new(widget.Editor),
		preRequestBody:  new(widget.Editor),
		postRequestBody: new(widget.Editor),
		section:         new(widget.Editor),
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		notification: &widgets.Notification{},

		copyResponseButton: widgets.NewFlatButton(theme, "Copy"),
		saveResponseButton: widgets.NewFlatButton(theme, "Save"),

		params: widgets.NewKeyValue(
			widgets.NewKeyValueItem("name", "John Doe", true),
			widgets.NewKeyValueItem("email", "", false),
		),
	}

	r.copyResponseButton.SetIcon(widgets.CopyIcon, widgets.FlatButtonIconEnd, 5)
	r.copyResponseButton.SetColor(theme.Palette.Bg, theme.Palette.Fg)
	r.copyResponseButton.MinWidth = unit.Dp(75)
	r.copyResponseButton.OnClicked = func() {
		clipboard.Write(clipboard.FmtText, []byte("text data"))
		notify.Send("Response copied to clipboard", time.Second*3)
	}

	r.saveResponseButton.SetIcon(widgets.SaveIcon, widgets.FlatButtonIconEnd, 5)
	r.saveResponseButton.SetColor(theme.Palette.Bg, theme.Palette.Fg)
	r.saveResponseButton.MinWidth = unit.Dp(75)
	r.saveResponseButton.OnClicked = func() {

	}

	r.sendButton = material.Button(theme, &r.sendClickable, "Send")

	r.requestTabs = widgets.NewTabs([]widgets.Tab{
		{Title: "Params"},
		{Title: "Body"},
		{Title: "Headers"},
		{Title: "Pre-req"},
		{Title: "Post-req"},
	}, nil)

	r.responseTabs = widgets.NewTabs([]widgets.Tab{
		{Title: "Body"},
		{Title: "Headers"},
	}, nil)

	r.methodDropDown = widgets.NewDropDown(theme,
		widgets.NewDropDownOption("GET"),
		widgets.NewDropDownOption("POST"),
		widgets.NewDropDownOption("PUT"),
		widgets.NewDropDownOption("PATCH"),
		widgets.NewDropDownOption("DELETE"),
		widgets.NewDropDownOption("HEAD"),
		widgets.NewDropDownOption("OPTION"),
	)
	r.methodDropDown.SetSize(image.Point{X: 150})

	r.preRequestDropDown = widgets.NewDropDown(theme,
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("Python Script"),
		widgets.NewDropDownOption("SSH Script"),
		widgets.NewDropDownOption("SSH Tunnel"),
		widgets.NewDropDownOption("Kubectl Tunnel"),
	)
	r.preRequestDropDown.SetSize(image.Point{X: 230})
	r.preRequestDropDown.SetBorder(widgets.Gray400, unit.Dp(1), unit.Dp(4))

	r.postRequestDropDown = widgets.NewDropDown(theme,
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("Python Script"),
		widgets.NewDropDownOption("SSH Script"),
	)

	r.postRequestDropDown.SetSize(image.Point{X: 230})
	r.postRequestDropDown.SetBorder(widgets.Gray400, unit.Dp(1), unit.Dp(4))

	r.requestBodyDropDown = widgets.NewDropDown(theme,
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("JSON"),
		widgets.NewDropDownOption("Text"),
		widgets.NewDropDownOption("XML"),
		widgets.NewDropDownOption("Form data"),
		widgets.NewDropDownOption("Binary"),
		widgets.NewDropDownOption("Urlencoded"),
	)
	r.requestBodyDropDown.SetSize(image.Point{X: 230})
	r.requestBodyDropDown.SetBorder(widgets.Gray400, unit.Dp(1), unit.Dp(4))

	r.address.SingleLine = true
	r.address.SetText("https://jsonplaceholder.typicode.com/comments")

	r.responseBody.SingleLine = false
	r.responseBody.ReadOnly = true

	return r
}

func (r *RestContainer) Submit() {
	method := r.methodDropDown.GetSelected().Text
	address := r.address.Text()

	r.sendButton.Text = "Cancel"
	r.loading = true
	r.resultUpdated = false
	defer func() {
		r.sendButton.Text = "Send"
		r.loading = false
		r.resultUpdated = false
	}()

	res, err := rest.DoRequest(&rest.Request{
		URL:     address,
		Method:  method,
		Headers: nil,
		Body:    nil,
	})
	if err != nil {
		r.responseBody.SetText(err.Error())
		return
	}

	dataStr := string(res.Body)
	if rest.IsJSON(dataStr) {
		var data map[string]interface{}
		if err := json.Unmarshal(res.Body, &data); err != nil {
			r.responseBody.SetText(err.Error())
			return
		}
		var err error
		dataStr, err = rest.PrettyJSON(res.Body)
		if err != nil {
			r.responseBody.SetText(err.Error())
			return
		}
	}

	// format response status
	r.resultStatus = fmt.Sprintf("%d %s, %s, %s", res.StatusCode, http.StatusText(res.StatusCode), res.TimePassed, humanize.Bytes(uint64(len(res.Body))))

	r.result = dataStr
}

func (r *RestContainer) requestBar(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        widgets.Gray400,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
			Spacing:   layout.SpaceEnd,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.methodDropDown.Layout(gtx)
			}),
			widgets.VerticalLine(40.0),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Editor(theme, r.address, "https://example.com").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if r.sendClickable.Clicked(gtx) {
						go r.Submit()
					}

					gtx.Constraints.Min.X = gtx.Dp(80)
					return r.sendButton.Layout(gtx)
				})
			}),
		)
	})
}

func (r *RestContainer) requestBodyLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, theme.TextSize, "Request body").Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.requestBodyDropDown.Layout(gtx)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(theme, r.requestBody, "").Layout(gtx)
			})
		}),
	)
}

func (r *RestContainer) requestPostReqLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, theme.TextSize, "Action to do after request").Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.postRequestDropDown.Layout(gtx)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(theme, r.postRequestBody, "").Layout(gtx)
			})
		}),
	)
}

func (r *RestContainer) requestPreReqLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, theme.TextSize, "Action to do before request").Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.preRequestDropDown.Layout(gtx)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(theme, r.preRequestBody, "").Layout(gtx)
			})
		}),
	)
}

func (r *RestContainer) requestLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.requestTabs.Layout(gtx, theme)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			switch r.requestTabs.Selected() {
			case 0:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.params.WithAddLayout(gtx, theme)
					})
				})

			case 1:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.requestBodyLayout(gtx, theme)
				})

			case 2:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme, theme.TextSize, "Headers").Layout(gtx)
				})

			case 3:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.requestPreReqLayout(gtx, theme)
				})

			case 4:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.requestPostReqLayout(gtx, theme)
				})
			}

			return layout.Dimensions{}
		}),
	)
}

func (r *RestContainer) responseLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if r.result == "" {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.LabelStyle{
				Text:     "No response available yet ;)",
				Color:    widgets.Gray600,
				TextSize: theme.TextSize,
				Shaper:   theme.Shaper,
			}
			l.Font.Typeface = theme.Face
			return l.Layout(gtx)
		})
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.responseTabs.Layout(gtx, theme)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.LabelStyle{
							Text:     r.resultStatus,
							Color:    widgets.LightGreen,
							TextSize: theme.TextSize,
							Shaper:   theme.Shaper,
						}
						l.Font.Typeface = theme.Face
						return l.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return r.copyResponseButton.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return r.saveResponseButton.Layout(gtx)
				}),
			)
		}),
		widgets.DrawLineFlex(widgets.Gray300, unit.Dp(1), unit.Dp(gtx.Constraints.Max.Y)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Editor(theme, r.responseBody, "").Layout(gtx)
		}),
	)
}

func (r *RestContainer) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Left:   unit.Dp(10),
				Top:    unit.Dp(10),
				Bottom: unit.Dp(5),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme, theme.TextSize, "Create user").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.requestBar(gtx, theme)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return r.split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return r.requestLayout(gtx, theme)
				},
				func(gtx layout.Context) layout.Dimensions {
					if r.loading {
						return material.Label(theme, theme.TextSize, "Loading...").Layout(gtx)
					} else {
						// update only once
						if !r.resultUpdated {
							r.responseBody.SetText(r.result)
							r.resultUpdated = true
						}
					}

					return r.responseLayout(gtx, theme)
				},
			)
		}),
	)
}
