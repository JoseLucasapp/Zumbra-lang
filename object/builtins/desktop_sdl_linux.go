//go:build linux && cgo

package builtins

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

typedef struct SDL_Window SDL_Window;
typedef struct SDL_Surface SDL_Surface;
typedef struct SDL_IOStream SDL_IOStream;
typedef struct SDL_Tray SDL_Tray;
typedef struct SDL_TrayMenu SDL_TrayMenu;
typedef struct SDL_TrayEntry SDL_TrayEntry;

typedef union ZSDL_Event { uint32_t type; uint8_t padding[128]; } ZSDL_Event;
typedef struct { uint32_t type,reserved; uint64_t timestamp; } ZSDL_CommonEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID; int32_t data1,data2; } ZSDL_WindowEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID,which,scancode,key; uint16_t mod,raw; bool down,repeat; } ZSDL_KeyboardEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID; const char *text; } ZSDL_TextInputEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID,which,state; float x,y,xrel,yrel; } ZSDL_MouseMotionEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID,which; uint8_t button; bool down; uint8_t clicks,padding; float x,y; } ZSDL_MouseButtonEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID,which; float x,y; uint32_t direction; float mouse_x,mouse_y; int32_t integer_x,integer_y; } ZSDL_MouseWheelEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID; float x,y; const char *source; const char *data; } ZSDL_DropEvent;

typedef struct {
    uint32_t type;
    uint64_t timestamp;
    uint32_t window_id;
    int64_t data1;
    int64_t data2;
    uint32_t key;
    uint16_t mod;
    int32_t button;
    int32_t clicks;
    float x;
    float y;
    float dx;
    float dy;
    char text[2048];
    char source[512];
} ZDesktopNativeEvent;

typedef bool (*PFN_SDL_Init)(uint32_t);
typedef void (*PFN_SDL_Quit)(void);
typedef const char *(*PFN_SDL_GetError)(void);
typedef bool (*PFN_SDL_SetAppMetadata)(const char*,const char*,const char*);
typedef SDL_Window *(*PFN_SDL_CreateWindow)(const char*,int,int,uint64_t);
typedef void (*PFN_SDL_DestroyWindow)(SDL_Window*);
typedef uint32_t (*PFN_SDL_GetWindowID)(SDL_Window*);
typedef bool (*PFN_SDL_ShowWindow)(SDL_Window*);
typedef bool (*PFN_SDL_HideWindow)(SDL_Window*);
typedef const char *(*PFN_SDL_GetWindowTitle)(SDL_Window*);
typedef bool (*PFN_SDL_SetWindowTitle)(SDL_Window*,const char*);
typedef bool (*PFN_SDL_GetWindowSize)(SDL_Window*,int*,int*);
typedef bool (*PFN_SDL_GetWindowSizeInPixels)(SDL_Window*,int*,int*);
typedef bool (*PFN_SDL_SetWindowSize)(SDL_Window*,int,int);
typedef bool (*PFN_SDL_GetWindowPosition)(SDL_Window*,int*,int*);
typedef bool (*PFN_SDL_SetWindowPosition)(SDL_Window*,int,int);
typedef bool (*PFN_SDL_SetWindowFullscreen)(SDL_Window*,bool);
typedef uint64_t (*PFN_SDL_GetWindowFlags)(SDL_Window*);
typedef bool (*PFN_SDL_MaximizeWindow)(SDL_Window*);
typedef bool (*PFN_SDL_MinimizeWindow)(SDL_Window*);
typedef bool (*PFN_SDL_RestoreWindow)(SDL_Window*);
typedef bool (*PFN_SDL_RaiseWindow)(SDL_Window*);
typedef float (*PFN_SDL_GetWindowDisplayScale)(SDL_Window*);
typedef float (*PFN_SDL_GetWindowPixelDensity)(SDL_Window*);
typedef bool (*PFN_SDL_PollEvent)(ZSDL_Event*);
typedef bool (*PFN_SDL_WaitEvent)(ZSDL_Event*);
typedef bool (*PFN_SDL_WaitEventTimeout)(ZSDL_Event*,int32_t);
typedef const char *(*PFN_SDL_GetKeyName)(uint32_t);
typedef bool (*PFN_SDL_SetClipboardText)(const char*);
typedef char *(*PFN_SDL_GetClipboardText)(void);
typedef void (*PFN_SDL_free)(void*);
typedef SDL_IOStream *(*PFN_SDL_IOFromFile)(const char*,const char*);
typedef SDL_Surface *(*PFN_SDL_LoadBMP_IO)(SDL_IOStream*,bool);
typedef void (*PFN_SDL_DestroySurface)(SDL_Surface*);
typedef bool (*PFN_SDL_SetWindowIcon)(SDL_Window*,SDL_Surface*);
typedef SDL_Tray *(*PFN_SDL_CreateTray)(SDL_Surface*,const char*);
typedef SDL_TrayMenu *(*PFN_SDL_CreateTrayMenu)(SDL_Tray*);
typedef SDL_TrayEntry *(*PFN_SDL_InsertTrayEntryAt)(SDL_TrayMenu*,int,const char*,uint32_t);
typedef void (*PFN_SDL_SetTrayEntryCallback)(SDL_TrayEntry*,void(*)(void*,SDL_TrayEntry*),void*);
typedef void (*PFN_SDL_SetTrayTooltip)(SDL_Tray*,const char*);
typedef void (*PFN_SDL_DestroyTray)(SDL_Tray*);

static void *zsdl_lib = NULL;
static PFN_SDL_Init p_Init; static PFN_SDL_Quit p_Quit; static PFN_SDL_GetError p_GetError;
static PFN_SDL_SetAppMetadata p_SetAppMetadata;
static PFN_SDL_CreateWindow p_CreateWindow; static PFN_SDL_DestroyWindow p_DestroyWindow; static PFN_SDL_GetWindowID p_GetWindowID;
static PFN_SDL_ShowWindow p_ShowWindow; static PFN_SDL_HideWindow p_HideWindow; static PFN_SDL_GetWindowTitle p_GetWindowTitle; static PFN_SDL_SetWindowTitle p_SetWindowTitle;
static PFN_SDL_GetWindowSize p_GetWindowSize; static PFN_SDL_GetWindowSizeInPixels p_GetWindowSizeInPixels; static PFN_SDL_SetWindowSize p_SetWindowSize;
static PFN_SDL_GetWindowPosition p_GetWindowPosition; static PFN_SDL_SetWindowPosition p_SetWindowPosition; static PFN_SDL_SetWindowFullscreen p_SetWindowFullscreen;
static PFN_SDL_GetWindowFlags p_GetWindowFlags; static PFN_SDL_MaximizeWindow p_MaximizeWindow; static PFN_SDL_MinimizeWindow p_MinimizeWindow; static PFN_SDL_RestoreWindow p_RestoreWindow; static PFN_SDL_RaiseWindow p_RaiseWindow;
static PFN_SDL_GetWindowDisplayScale p_GetWindowDisplayScale; static PFN_SDL_GetWindowPixelDensity p_GetWindowPixelDensity;
static PFN_SDL_PollEvent p_PollEvent; static PFN_SDL_WaitEvent p_WaitEvent; static PFN_SDL_WaitEventTimeout p_WaitEventTimeout; static PFN_SDL_GetKeyName p_GetKeyName;
static PFN_SDL_SetClipboardText p_SetClipboardText; static PFN_SDL_GetClipboardText p_GetClipboardText; static PFN_SDL_free p_free;
static PFN_SDL_IOFromFile p_IOFromFile; static PFN_SDL_LoadBMP_IO p_LoadBMP_IO; static PFN_SDL_DestroySurface p_DestroySurface; static PFN_SDL_SetWindowIcon p_SetWindowIcon;
static PFN_SDL_CreateTray p_CreateTray; static PFN_SDL_CreateTrayMenu p_CreateTrayMenu; static PFN_SDL_InsertTrayEntryAt p_InsertTrayEntryAt; static PFN_SDL_SetTrayEntryCallback p_SetTrayEntryCallback; static PFN_SDL_SetTrayTooltip p_SetTrayTooltip; static PFN_SDL_DestroyTray p_DestroyTray;
static char zsdl_error[512];

#define LOAD_REQ(name) do { p_##name = (PFN_SDL_##name)dlsym(zsdl_lib,"SDL_" #name); if(!p_##name){snprintf(zsdl_error,sizeof(zsdl_error),"missing SDL3 symbol SDL_%s",#name);return false;} } while(0)
#define LOAD_OPT(name) do { p_##name = (PFN_SDL_##name)dlsym(zsdl_lib,"SDL_" #name); } while(0)

static bool zsdl_load(void) {
    if (zsdl_lib) return true;
    const char *names[] = {"libSDL3.so.0","libSDL3.so",NULL};
    for(int i=0;names[i];i++){zsdl_lib=dlopen(names[i],RTLD_NOW|RTLD_LOCAL);if(zsdl_lib)break;}
    if(!zsdl_lib){snprintf(zsdl_error,sizeof(zsdl_error),"SDL3 shared library was not found (install libsdl3-0 or build SDL3)");return false;}
    LOAD_REQ(Init); LOAD_REQ(Quit); LOAD_REQ(GetError); LOAD_OPT(SetAppMetadata);
    LOAD_REQ(CreateWindow); LOAD_REQ(DestroyWindow); LOAD_REQ(GetWindowID); LOAD_REQ(ShowWindow); LOAD_REQ(HideWindow); LOAD_REQ(GetWindowTitle); LOAD_REQ(SetWindowTitle);
    LOAD_REQ(GetWindowSize); LOAD_REQ(GetWindowSizeInPixels); LOAD_REQ(SetWindowSize); LOAD_REQ(GetWindowPosition); LOAD_REQ(SetWindowPosition); LOAD_REQ(SetWindowFullscreen); LOAD_REQ(GetWindowFlags);
    LOAD_REQ(MaximizeWindow); LOAD_REQ(MinimizeWindow); LOAD_REQ(RestoreWindow); LOAD_REQ(RaiseWindow); LOAD_REQ(GetWindowDisplayScale); LOAD_REQ(GetWindowPixelDensity);
    LOAD_REQ(PollEvent); LOAD_REQ(WaitEvent); LOAD_REQ(WaitEventTimeout); LOAD_REQ(GetKeyName); LOAD_REQ(SetClipboardText); LOAD_REQ(GetClipboardText); LOAD_REQ(free);
    LOAD_OPT(IOFromFile); LOAD_OPT(LoadBMP_IO); LOAD_OPT(DestroySurface); LOAD_OPT(SetWindowIcon);
    LOAD_OPT(CreateTray); LOAD_OPT(CreateTrayMenu); LOAD_OPT(InsertTrayEntryAt); LOAD_OPT(SetTrayEntryCallback); LOAD_OPT(SetTrayTooltip); LOAD_OPT(DestroyTray);
    return true;
}
static const char *zsdl_last_error(void){ if(zsdl_error[0])return zsdl_error; if(p_GetError){const char*e=p_GetError();if(e&&*e)return e;} return "unknown SDL3 error"; }
static bool zsdl_init(const char*name,const char*version,const char*identifier){ if(!zsdl_load())return false; if(p_SetAppMetadata)p_SetAppMetadata(name,version,identifier); if(!p_Init(0x00000020u)){snprintf(zsdl_error,sizeof(zsdl_error),"%s",p_GetError());return false;} return true; }
static void zsdl_quit(void){if(p_Quit)p_Quit();}
static SDL_Window *zsdl_create_window(const char*t,int w,int h,uint64_t flags){return p_CreateWindow(t,w,h,flags);}
static void zsdl_destroy_window(SDL_Window*w){if(w)p_DestroyWindow(w);} static uint32_t zsdl_window_id(SDL_Window*w){return p_GetWindowID(w);}
static bool zsdl_show(SDL_Window*w){return p_ShowWindow(w);} static bool zsdl_hide(SDL_Window*w){return p_HideWindow(w);}
static const char*zsdl_title(SDL_Window*w){return p_GetWindowTitle(w);} static bool zsdl_set_title(SDL_Window*w,const char*t){return p_SetWindowTitle(w,t);}
static bool zsdl_size(SDL_Window*w,int*a,int*b){return p_GetWindowSize(w,a,b);} static bool zsdl_pixel_size(SDL_Window*w,int*a,int*b){return p_GetWindowSizeInPixels(w,a,b);} static bool zsdl_set_size(SDL_Window*w,int a,int b){return p_SetWindowSize(w,a,b);}
static bool zsdl_position(SDL_Window*w,int*a,int*b){return p_GetWindowPosition(w,a,b);} static bool zsdl_set_position(SDL_Window*w,int a,int b){return p_SetWindowPosition(w,a,b);}
static bool zsdl_set_fullscreen(SDL_Window*w,bool v){return p_SetWindowFullscreen(w,v);} static uint64_t zsdl_flags(SDL_Window*w){return p_GetWindowFlags(w);}
static bool zsdl_maximize(SDL_Window*w){return p_MaximizeWindow(w);} static bool zsdl_minimize(SDL_Window*w){return p_MinimizeWindow(w);} static bool zsdl_restore(SDL_Window*w){return p_RestoreWindow(w);} static bool zsdl_raise(SDL_Window*w){return p_RaiseWindow(w);}
static float zsdl_display_scale(SDL_Window*w){return p_GetWindowDisplayScale(w);} static float zsdl_pixel_density(SDL_Window*w){return p_GetWindowPixelDensity(w);}
static bool zsdl_set_clipboard(const char*t){return p_SetClipboardText(t);} static char*zsdl_get_clipboard(void){return p_GetClipboardText();} static void zsdl_free_text(void*p){if(p)p_free(p);}
static bool zsdl_set_icon(SDL_Window*w,const char*path){if(!p_IOFromFile||!p_LoadBMP_IO||!p_SetWindowIcon||!p_DestroySurface){snprintf(zsdl_error,sizeof(zsdl_error),"SDL3 BMP icon APIs are unavailable");return false;}SDL_IOStream*io=p_IOFromFile(path,"rb");if(!io)return false;SDL_Surface*s=p_LoadBMP_IO(io,true);if(!s)return false;bool ok=p_SetWindowIcon(w,s);p_DestroySurface(s);return ok;}

#define ZSDL_TRAY_QUEUE 128
static char *zsdl_tray_queue[ZSDL_TRAY_QUEUE]; static int zsdl_tray_head=0,zsdl_tray_tail=0;
static void zsdl_tray_callback(void*userdata,SDL_TrayEntry*entry){(void)entry;const char*id=(const char*)userdata;int next=(zsdl_tray_tail+1)%ZSDL_TRAY_QUEUE;if(next==zsdl_tray_head)return;const char*value=id?id:"";size_t n=strlen(value)+1;char*copy=(char*)malloc(n);if(!copy)return;memcpy(copy,value,n);zsdl_tray_queue[zsdl_tray_tail]=copy;zsdl_tray_tail=next;}
static char *zsdl_tray_poll(void){if(zsdl_tray_head==zsdl_tray_tail)return NULL;char*v=zsdl_tray_queue[zsdl_tray_head];zsdl_tray_head=(zsdl_tray_head+1)%ZSDL_TRAY_QUEUE;return v;}
static SDL_Surface *zsdl_load_bmp(const char*path){if(!path||!*path||!p_IOFromFile||!p_LoadBMP_IO)return NULL;SDL_IOStream*io=p_IOFromFile(path,"rb");if(!io)return NULL;return p_LoadBMP_IO(io,true);}
static SDL_Tray *zsdl_create_tray(const char*icon,const char*tooltip){if(!p_CreateTray)return NULL;SDL_Surface*s=zsdl_load_bmp(icon);SDL_Tray*t=p_CreateTray(s,tooltip);if(s&&p_DestroySurface)p_DestroySurface(s);return t;}
static SDL_TrayMenu *zsdl_create_tray_menu(SDL_Tray*t){if(!t||!p_CreateTrayMenu)return NULL;return p_CreateTrayMenu(t);}
static bool zsdl_tray_add(SDL_TrayMenu*m,const char*id,const char*label){if(!m||!p_InsertTrayEntryAt||!p_SetTrayEntryCallback)return false;SDL_TrayEntry*e=p_InsertTrayEntryAt(m,-1,label,1u);if(!e)return false;size_t n=strlen(id)+1;char*copy=(char*)malloc(n);if(!copy)return false;memcpy(copy,id,n);p_SetTrayEntryCallback(e,zsdl_tray_callback,copy);return true;}
static bool zsdl_tray_tooltip(SDL_Tray*t,const char*v){if(!p_SetTrayTooltip)return false;p_SetTrayTooltip(t,v);return true;} static void zsdl_destroy_tray(SDL_Tray*t){if(t&&p_DestroyTray)p_DestroyTray(t);}

static void zsdl_copy(char*dst,size_t n,const char*src){if(!dst||n==0)return;if(!src){dst[0]=0;return;}snprintf(dst,n,"%s",src);}
static bool zsdl_poll(int timeout,ZDesktopNativeEvent*out){memset(out,0,sizeof(*out));char*tray=zsdl_tray_poll();if(tray){out->type=0xF001;zsdl_copy(out->text,sizeof(out->text),tray);free(tray);return true;}ZSDL_Event e;bool ok=timeout<0?p_WaitEvent(&e):(timeout==0?p_PollEvent(&e):p_WaitEventTimeout(&e,timeout));if(!ok)return false;ZSDL_CommonEvent*c=(ZSDL_CommonEvent*)&e;out->type=c->type;out->timestamp=c->timestamp;
    if((c->type>=0x202&&c->type<=0x220)){ZSDL_WindowEvent*w=(ZSDL_WindowEvent*)&e;out->window_id=w->windowID;out->data1=w->data1;out->data2=w->data2;}
    else if(c->type==0x300||c->type==0x301){ZSDL_KeyboardEvent*k=(ZSDL_KeyboardEvent*)&e;out->window_id=k->windowID;out->key=k->key;out->mod=k->mod;out->data1=k->down;out->data2=k->repeat;const char*n=p_GetKeyName(k->key);zsdl_copy(out->text,sizeof(out->text),n);}
    else if(c->type==0x303){ZSDL_TextInputEvent*t=(ZSDL_TextInputEvent*)&e;out->window_id=t->windowID;zsdl_copy(out->text,sizeof(out->text),t->text);}
    else if(c->type==0x400){ZSDL_MouseMotionEvent*m=(ZSDL_MouseMotionEvent*)&e;out->window_id=m->windowID;out->x=m->x;out->y=m->y;out->dx=m->xrel;out->dy=m->yrel;out->data1=m->state;}
    else if(c->type==0x401||c->type==0x402){ZSDL_MouseButtonEvent*b=(ZSDL_MouseButtonEvent*)&e;out->window_id=b->windowID;out->button=b->button;out->clicks=b->clicks;out->x=b->x;out->y=b->y;out->data1=b->down;}
    else if(c->type==0x403){ZSDL_MouseWheelEvent*w=(ZSDL_MouseWheelEvent*)&e;out->window_id=w->windowID;out->x=w->x;out->y=w->y;out->data1=w->direction;}
    else if(c->type>=0x1000&&c->type<=0x1004){ZSDL_DropEvent*d=(ZSDL_DropEvent*)&e;out->window_id=d->windowID;out->x=d->x;out->y=d->y;zsdl_copy(out->text,sizeof(out->text),d->data);zsdl_copy(out->source,sizeof(out->source),d->source);}
    return true;}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"zumbra/object"
)

type sdlDesktopBackend struct {
	mu            sync.Mutex
	windows       map[int64]*sdlWindow
	closed        bool
	nextSynthetic int64
}
type sdlWindow struct {
	backend *sdlDesktopBackend
	mu      sync.RWMutex
	handle  *C.SDL_Window
	id      int64
	open    bool
}
type sdlTray struct {
	mu     sync.RWMutex
	handle *C.SDL_Tray
	menu   *C.SDL_TrayMenu
	open   bool
}

func newPlatformDesktopBackend(options map[string]object.Object) (object.DesktopBackend, error) {
	if optionBool(options, "headless", false) || os.Getenv("ZUMBRA_DESKTOP_HEADLESS") == "1" {
		return newHeadlessDesktopBackend(), nil
	}
	runtime.LockOSThread()
	name := optionString(options, "name", "Zumbra")
	version := optionString(options, "version", "")
	identifier := optionString(options, "identifier", "dev.zumbra.app")
	cn, cv, ci := C.CString(name), C.CString(version), C.CString(identifier)
	defer C.free(unsafe.Pointer(cn))
	defer C.free(unsafe.Pointer(cv))
	defer C.free(unsafe.Pointer(ci))
	if !bool(C.zsdl_init(cn, cv, ci)) {
		runtime.UnlockOSThread()
		return nil, errors.New(C.GoString(C.zsdl_last_error()))
	}
	return &sdlDesktopBackend{windows: map[int64]*sdlWindow{}}, nil
}
func (b *sdlDesktopBackend) Name() string { return "sdl3" }
func (b *sdlDesktopBackend) CreateWindow(options map[string]object.Object) (object.DesktopWindowRuntime, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errors.New("desktop backend is closed")
	}
	title := optionString(options, "title", "Zumbra")
	w := optionInt(options, "width", 800)
	h := optionInt(options, "height", 600)
	if w < 1 || h < 1 {
		return nil, errors.New("window width and height must be positive")
	}
	var flags uint64
	if optionBool(options, "fullscreen", false) {
		flags |= 0x1
	}
	if optionBool(options, "hidden", false) {
		flags |= 0x8
	}
	if optionBool(options, "borderless", false) {
		flags |= 0x10
	}
	if optionBool(options, "resizable", true) {
		flags |= 0x20
	}
	if optionBool(options, "highDPI", true) {
		flags |= 0x2000
	}
	if optionBool(options, "alwaysOnTop", false) {
		flags |= 0x8000
	}
	if optionBool(options, "transparent", false) {
		flags |= 0x40000000
	}
	ct := C.CString(title)
	defer C.free(unsafe.Pointer(ct))
	handle := C.zsdl_create_window(ct, C.int(w), C.int(h), C.uint64_t(flags))
	if handle == nil {
		return nil, errors.New(C.GoString(C.zsdl_last_error()))
	}
	id := int64(C.zsdl_window_id(handle))
	window := &sdlWindow{backend: b, handle: handle, id: id, open: true}
	b.windows[id] = window
	if icon := optionString(options, "icon", ""); icon != "" {
		_ = window.SetIcon(icon)
	}
	return window, nil
}
func (b *sdlDesktopBackend) PollEvent(timeoutMS int64) (*object.DesktopEvent, error) {
	var e C.ZDesktopNativeEvent
	if !bool(C.zsdl_poll(C.int(timeoutMS), &e)) {
		return nil, nil
	}
	kind, data := sdlEvent(int64(e._type), &e)
	return &object.DesktopEvent{Type: kind, WindowID: int64(e.window_id), Timestamp: int64(e.timestamp), Data: data}, nil
}
func sdlEvent(t int64, e *C.ZDesktopNativeEvent) (string, map[string]object.Object) {
	d := map[string]object.Object{}
	switch t {
	case 0x100:
		return "quit", d
	case 0x101:
		return "terminating", d
	case 0x102:
		return "low_memory", d
	case 0x103:
		return "will_enter_background", d
	case 0x104:
		return "did_enter_background", d
	case 0x105:
		return "will_enter_foreground", d
	case 0x106:
		return "did_enter_foreground", d
	case 0x109:
		return "system_theme_changed", d
	case 0x202:
		return "window_shown", d
	case 0x203:
		return "window_hidden", d
	case 0x204:
		return "window_exposed", d
	case 0x205:
		d["x"] = NewInteger(int64(e.data1))
		d["y"] = NewInteger(int64(e.data2))
		return "window_moved", d
	case 0x206:
		d["width"] = NewInteger(int64(e.data1))
		d["height"] = NewInteger(int64(e.data2))
		return "window_resized", d
	case 0x207:
		d["width"] = NewInteger(int64(e.data1))
		d["height"] = NewInteger(int64(e.data2))
		return "window_pixel_size_changed", d
	case 0x209:
		return "window_minimized", d
	case 0x20a:
		return "window_maximized", d
	case 0x20b:
		return "window_restored", d
	case 0x20c:
		return "window_mouse_enter", d
	case 0x20d:
		return "window_mouse_leave", d
	case 0x20e:
		return "window_focus_gained", d
	case 0x20f:
		return "window_focus_lost", d
	case 0x210:
		return "window_close_requested", d
	case 0x219:
		return "window_enter_fullscreen", d
	case 0x21a:
		return "window_leave_fullscreen", d
	case 0x21b:
		return "window_closed", d
	case 0x300, 0x301:
		d["key"] = NewString(C.GoString(&e.text[0]))
		d["keycode"] = NewInteger(int64(e.key))
		d["modifiers"] = NewInteger(int64(e.mod))
		d["repeat"] = NewBoolean(e.data2 != 0)
		d["shortcut"] = NewString(shortcutFromSDL(C.GoString(&e.text[0]), uint16(e.mod)))
		if t == 0x300 {
			return "key_down", d
		}
		return "key_up", d
	case 0x303:
		d["text"] = NewString(C.GoString(&e.text[0]))
		return "text_input", d
	case 0x400:
		d["x"] = NewFloat(float64(e.x))
		d["y"] = NewFloat(float64(e.y))
		d["dx"] = NewFloat(float64(e.dx))
		d["dy"] = NewFloat(float64(e.dy))
		return "mouse_move", d
	case 0x401, 0x402:
		d["button"] = NewInteger(int64(e.button))
		d["clicks"] = NewInteger(int64(e.clicks))
		d["x"] = NewFloat(float64(e.x))
		d["y"] = NewFloat(float64(e.y))
		if t == 0x401 {
			return "mouse_down", d
		}
		return "mouse_up", d
	case 0x403:
		d["x"] = NewFloat(float64(e.x))
		d["y"] = NewFloat(float64(e.y))
		return "mouse_wheel", d
	case 0x900:
		return "clipboard_updated", d
	case 0x1000:
		d["path"] = NewString(C.GoString(&e.text[0]))
		d["source"] = NewString(C.GoString(&e.source[0]))
		d["x"] = NewFloat(float64(e.x))
		d["y"] = NewFloat(float64(e.y))
		return "drop_file", d
	case 0x1001:
		d["text"] = NewString(C.GoString(&e.text[0]))
		return "drop_text", d
	case 0x1002:
		return "drop_begin", d
	case 0x1003:
		return "drop_complete", d
	case 0xF001:
		d["id"] = NewString(C.GoString(&e.text[0]))
		return "tray", d
	default:
		d["nativeType"] = NewInteger(t)
		return "unknown", d
	}
}
func shortcutFromSDL(key string, mod uint16) string {
	parts := []string{}
	if mod&0x00c0 != 0 {
		parts = append(parts, "ctrl")
	}
	if mod&0x0003 != 0 {
		parts = append(parts, "shift")
	}
	if mod&0x0300 != 0 {
		parts = append(parts, "alt")
	}
	if mod&0x0c00 != 0 {
		parts = append(parts, "super")
	}
	key = strings.ToLower(strings.TrimSpace(key))
	if key != "" {
		parts = append(parts, key)
	}
	return strings.Join(parts, "+")
}
func (b *sdlDesktopBackend) SetClipboard(text string) error {
	c := C.CString(text)
	defer C.free(unsafe.Pointer(c))
	if !bool(C.zsdl_set_clipboard(c)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (b *sdlDesktopBackend) Clipboard() (string, error) {
	p := C.zsdl_get_clipboard()
	if p == nil {
		return "", errors.New(C.GoString(C.zsdl_last_error()))
	}
	defer C.zsdl_free_text(unsafe.Pointer(p))
	return C.GoString(p), nil
}
func (b *sdlDesktopBackend) PickFile(options map[string]object.Object) ([]string, error) {
	args := []string{"--file-selection"}
	if optionBool(options, "multiple", false) {
		args = append(args, "--multiple", "--separator=\n")
	}
	if title := optionString(options, "title", ""); title != "" {
		args = append(args, "--title="+title)
	}
	if path := optionString(options, "defaultPath", ""); path != "" {
		args = append(args, "--filename="+path)
	}
	out, err := runDialog("zenity", args, []string{"kdialog", "--getopenfilename", optionString(options, "defaultPath", "")})
	if err != nil {
		return nil, err
	}
	lines := []string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
func (b *sdlDesktopBackend) PickFolder(options map[string]object.Object) (string, error) {
	args := []string{"--file-selection", "--directory"}
	if title := optionString(options, "title", ""); title != "" {
		args = append(args, "--title="+title)
	}
	if path := optionString(options, "defaultPath", ""); path != "" {
		args = append(args, "--filename="+path)
	}
	out, err := runDialog("zenity", args, []string{"kdialog", "--getexistingdirectory", optionString(options, "defaultPath", "")})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
func runDialog(primary string, args, alternate []string) (string, error) {
	if path, err := exec.LookPath(primary); err == nil {
		out, e := exec.Command(path, args...).Output()
		if e == nil {
			return string(out), nil
		}
		if ee, ok := e.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return "", nil
		}
		return "", e
	}
	if len(alternate) > 0 {
		if path, err := exec.LookPath(alternate[0]); err == nil {
			out, e := exec.Command(path, alternate[1:]...).Output()
			if e == nil {
				return string(out), nil
			}
			if ee, ok := e.(*exec.ExitError); ok && ee.ExitCode() == 1 {
				return "", nil
			}
			return "", e
		}
	}
	return "", errors.New("no desktop file dialog provider found; install zenity or kdialog")
}
func (b *sdlDesktopBackend) Notify(options map[string]object.Object) error {
	path, err := exec.LookPath("notify-send")
	if err != nil {
		return errors.New("notify-send is required for desktop notifications")
	}
	args := []string{}
	if icon := optionString(options, "icon", ""); icon != "" {
		args = append(args, "--icon", icon)
	}
	if urgency := optionString(options, "urgency", ""); urgency != "" {
		args = append(args, "--urgency", urgency)
	}
	args = append(args, optionString(options, "title", "Zumbra"), optionString(options, "body", ""))
	return exec.Command(path, args...).Run()
}
func (b *sdlDesktopBackend) CreateTray(options map[string]object.Object) (object.DesktopTrayRuntime, error) {
	icon := optionString(options, "icon", "")
	tooltip := optionString(options, "tooltip", optionString(options, "title", "Zumbra"))
	ci, ct := C.CString(icon), C.CString(tooltip)
	defer C.free(unsafe.Pointer(ci))
	defer C.free(unsafe.Pointer(ct))
	h := C.zsdl_create_tray(ci, ct)
	if h == nil {
		return nil, errors.New("SDL3 system tray is unavailable: " + C.GoString(C.zsdl_last_error()))
	}
	menu := C.zsdl_create_tray_menu(h)
	if menu == nil {
		C.zsdl_destroy_tray(h)
		return nil, errors.New("SDL3 could not create the system tray menu")
	}
	return &sdlTray{handle: h, menu: menu, open: true}, nil
}
func (b *sdlDesktopBackend) Paths() map[string]string { return desktopPaths() }
func (b *sdlDesktopBackend) OpenExternal(target string) error {
	if target == "" {
		return errors.New("target cannot be empty")
	}
	path, err := exec.LookPath("xdg-open")
	if err != nil {
		return err
	}
	return exec.Command(path, target).Start()
}
func (b *sdlDesktopBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	for _, w := range b.windows {
		_ = w.Close()
	}
	C.zsdl_quit()
	return nil
}
func (w *sdlWindow) ensure() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if !w.open || w.handle == nil {
		return errors.New("window is closed")
	}
	return nil
}
func (w *sdlWindow) ID() int64 { return w.id }
func (w *sdlWindow) Show() error {
	if err := w.ensure(); err != nil {
		return err
	}
	if !bool(C.zsdl_show(w.handle)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) Hide() error {
	if err := w.ensure(); err != nil {
		return err
	}
	if !bool(C.zsdl_hide(w.handle)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open {
		return nil
	}
	w.open = false
	if w.handle != nil {
		C.zsdl_destroy_window(w.handle)
		w.handle = nil
	}
	return nil
}
func (w *sdlWindow) IsOpen() bool { w.mu.RLock(); defer w.mu.RUnlock(); return w.open }
func (w *sdlWindow) Title() string {
	if !w.IsOpen() {
		return ""
	}
	return C.GoString(C.zsdl_title(w.handle))
}
func (w *sdlWindow) SetTitle(v string) error {
	if err := w.ensure(); err != nil {
		return err
	}
	c := C.CString(v)
	defer C.free(unsafe.Pointer(c))
	if !bool(C.zsdl_set_title(w.handle, c)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) Size() (int64, int64) {
	var a, b C.int
	if !w.IsOpen() || !bool(C.zsdl_size(w.handle, &a, &b)) {
		return 0, 0
	}
	return int64(a), int64(b)
}
func (w *sdlWindow) PixelSize() (int64, int64) {
	var a, b C.int
	if !w.IsOpen() || !bool(C.zsdl_pixel_size(w.handle, &a, &b)) {
		return 0, 0
	}
	return int64(a), int64(b)
}
func (w *sdlWindow) SetSize(a, b int64) error {
	if a < 1 || b < 1 {
		return errors.New("window width and height must be positive")
	}
	if err := w.ensure(); err != nil {
		return err
	}
	if !bool(C.zsdl_set_size(w.handle, C.int(a), C.int(b))) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) Position() (int64, int64) {
	var a, b C.int
	if !w.IsOpen() || !bool(C.zsdl_position(w.handle, &a, &b)) {
		return 0, 0
	}
	return int64(a), int64(b)
}
func (w *sdlWindow) SetPosition(a, b int64) error {
	if err := w.ensure(); err != nil {
		return err
	}
	if !bool(C.zsdl_set_position(w.handle, C.int(a), C.int(b))) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) Fullscreen() bool {
	if !w.IsOpen() {
		return false
	}
	return uint64(C.zsdl_flags(w.handle))&1 != 0
}
func (w *sdlWindow) SetFullscreen(v bool) error {
	if err := w.ensure(); err != nil {
		return err
	}
	if !bool(C.zsdl_set_fullscreen(w.handle, C.bool(v))) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) Maximize() error {
	if err := w.ensure(); err != nil {
		return err
	}
	if !bool(C.zsdl_maximize(w.handle)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) Minimize() error {
	if err := w.ensure(); err != nil {
		return err
	}
	if !bool(C.zsdl_minimize(w.handle)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) Restore() error {
	if err := w.ensure(); err != nil {
		return err
	}
	if !bool(C.zsdl_restore(w.handle)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) Focus() error {
	if err := w.ensure(); err != nil {
		return err
	}
	if !bool(C.zsdl_raise(w.handle)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (w *sdlWindow) DisplayScale() float64 {
	if !w.IsOpen() {
		return 0
	}
	return float64(C.zsdl_display_scale(w.handle))
}
func (w *sdlWindow) PixelDensity() float64 {
	if !w.IsOpen() {
		return 0
	}
	return float64(C.zsdl_pixel_density(w.handle))
}
func (w *sdlWindow) SetIcon(path string) error {
	if err := validateDesktopPath(path); err != nil {
		return err
	}
	if err := w.ensure(); err != nil {
		return err
	}
	c := C.CString(path)
	defer C.free(unsafe.Pointer(c))
	if !bool(C.zsdl_set_icon(w.handle, c)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
}
func (t *sdlTray) Add(id, label string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.open {
		return errors.New("tray is closed")
	}
	ci, cl := C.CString(id), C.CString(label)
	defer C.free(unsafe.Pointer(ci))
	defer C.free(unsafe.Pointer(cl))
	if !bool(C.zsdl_tray_add(t.menu, ci, cl)) {
		return errors.New("could not add SDL3 tray entry")
	}
	return nil
}
func (t *sdlTray) SetTooltip(v string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.open {
		return errors.New("tray is closed")
	}
	c := C.CString(v)
	defer C.free(unsafe.Pointer(c))
	if !bool(C.zsdl_tray_tooltip(t.handle, c)) {
		return errors.New("could not update SDL3 tray tooltip")
	}
	return nil
}
func (t *sdlTray) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.open {
		return nil
	}
	t.open = false
	C.zsdl_destroy_tray(t.handle)
	t.handle = nil
	return nil
}
func (t *sdlTray) IsOpen() bool { t.mu.RLock(); defer t.mu.RUnlock(); return t.open }

type osDesktopProcess struct {
	mu      sync.RWMutex
	cmd     *exec.Cmd
	done    chan struct{}
	exit    int64
	err     error
	running bool
	pid     int64
}

func startDesktopProcess(command string, args []string, options map[string]object.Object) (object.DesktopProcessRuntime, error) {
	if command == "" {
		return nil, errors.New("command cannot be empty")
	}
	cmd := exec.Command(command, args...)
	if cwd := optionString(options, "cwd", ""); cwd != "" {
		cmd.Dir = cwd
	}
	if envObject, ok := options["env"].(*object.Dict); ok {
		cmd.Env = os.Environ()
		for _, pair := range envObject.Pairs {
			key, kok := pair.Key.(*object.String)
			value, vok := pair.Value.(*object.String)
			if kok && vok {
				cmd.Env = append(cmd.Env, key.Value+"="+value.Value)
			}
		}
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &osDesktopProcess{cmd: cmd, done: make(chan struct{}), running: true, pid: int64(cmd.Process.Pid)}
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.err = err
		p.running = false
		if cmd.ProcessState != nil {
			p.exit = int64(cmd.ProcessState.ExitCode())
		}
		p.mu.Unlock()
		close(p.done)
	}()
	return p, nil
}
func (p *osDesktopProcess) PID() int64 { return p.pid }
func (p *osDesktopProcess) Wait() (int64, error) {
	<-p.done
	p.mu.RLock()
	defer p.mu.RUnlock()
	if _, ok := p.err.(*exec.ExitError); ok {
		return p.exit, nil
	}
	return p.exit, p.err
}
func (p *osDesktopProcess) Kill() error {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()
	if !running {
		return nil
	}
	return p.cmd.Process.Kill()
}
func (p *osDesktopProcess) Running() bool { p.mu.RLock(); defer p.mu.RUnlock(); return p.running }

func parseProcessArgs(value object.Object) ([]string, *object.Error) {
	array, ok := value.(*object.Array)
	if !ok {
		return nil, NewError("desktopSpawn arguments must be array")
	}
	result := make([]string, len(array.Elements))
	for index, item := range array.Elements {
		text, ok := item.(*object.String)
		if !ok {
			return nil, NewError("desktopSpawn argument %d must be string", index)
		}
		result[index] = text.Value
	}
	return result, nil
}
func _unusedDesktopSDLImports() { _, _ = fmt.Sprint, strconv.Itoa }
