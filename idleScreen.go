package main

import (
        "fmt"
        "time"
)

type idleScreen struct{
        BaseScreen[idleScreen]
        modal *time.Timer
}

// Instantiation must include setting "self" or callbacks won't work!
func NewidleScreen() *idleScreen {
         s:= &idleScreen{}
         s.self=s
         return s
 }

func (screen *idleScreen) Init() error  {
        // MUST DO for instantiator??
        screen.self = screen
        screen.modal=nil
        return nil
}

func (screen *idleScreen) HandleEvent(evt UIEvent) error  {
        fmt.Printf("Got event %+v\n",evt)
        switch evt.Event {
                case Event_Encoderknob:
                       video_updateknob(evt)
                    break
                case Event_Button:
                        if (screen.modal != nil) {
                           screen.BaseScreen.RemoveTimer(screen.modal)
                           screen.Draw()
                           screen.modal = nil
                        } else {
                                uiEvent <- UIEvent { Event: Event_Alert, Name:evt.Name }
                        }
                        break

                case Event_Alert:
                   screen.Alert(evt)
                   screen.modal = screen.BaseScreen.AddTimer(10* time.Second,ClearAlert,0)
                   break
        }
        return nil
}

func ClearAlert(screen *idleScreen, x any) {
    screen.modal = nil
    screen.Draw()
    video_update()
}
func (screen *idleScreen) Alert(evt UIEvent) {
	dc.SetRGB(1, 1, 0)
	dc.DrawRoundedRectangle(50, 50, float64(WIDTH-100), float64(HEIGHT-100),25.0)
	dc.Fill()


    pngImage, err, w, h := loadAndScaleImage("alert.png",WIDTH/4,0)
    if (err != nil) {
            fmt.Errorf("Error loading image %w\n",err)
    }
    dc.DrawImage(pngImage, 70, (HEIGHT-h)/2)

	dc.SetRGB(1, 0, 0)
    setFontSize(42)

    // SHould we...?
    // DrawStringWrapped(s string, x, y, ax, ay, width, lineSpacing float64, align Align)
    // or func (dc *Context) MeasureMultilineString(s string, lineSpacing float64) (width, height float64)
    // or func (dc *Context) MeasureString(s string) (w, h float64)
    // or  DrawStringWrapped(s string, x, y, ax, ay, width, lineSpacing float64, align Align)

	dc.DrawString(evt.Name, float64(70+w), 200)

    video_update()

}


func (screen *idleScreen) Close() error  {
        return screen.BaseScreen.Close()
}

func (screen *idleScreen) Draw()  {
    // Background
	dc.SetRGB(0, 0.5, 0)
	dc.DrawRectangle(0, 0, 1024, 600)
	dc.Fill()

	dc.SetRGB(1, 1, 1)

            setFontSize(64)
            h := float64(110)
            dc.DrawStringAnchored("Room Available", float64(WIDTH/2), h, 0.5, 0.5)
            setFontSize(32)
            h+=50
            dc.DrawStringAnchored("Swipe fob to use room", float64(WIDTH/2), h, 0.5, 0.5)





    /* Lower Banner */

    setFontSize(42)
	dc.SetRGB(0.3, 0.5, 0.3)
	now := time.Now()
    TIMEYPOS:=float64(HEIGHT-76)
	dc.DrawRectangle(0, TIMEYPOS, float64(WIDTH), 66)
	dc.Fill()
	dc.SetRGB(0, 0.25, 0)

	now = time.Now()
    formattedDateTime := now.Format("Jan 2 3:04pm")
	dc.DrawStringAnchored(formattedDateTime, float64(WIDTH/2), TIMEYPOS+30, 0.5, 0.5)


    // Top User Banner
	dc.SetRGB(0.3, 0.5, 0.3)
	dc.DrawRectangle(0, 10, float64(WIDTH), 66)
	dc.Fill()
	dc.SetRGB(0.0, 0.0, 0.0)

    video_update()
}
