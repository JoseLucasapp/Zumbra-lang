//go:build linux && cgo

package builtins

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <unistd.h>
#include <ctype.h>
#include <stdio.h>
#include <math.h>

typedef struct SDL_Window SDL_Window;
typedef struct SDL_Surface SDL_Surface;
typedef struct SDL_IOStream SDL_IOStream;
typedef struct SDL_Tray SDL_Tray;
typedef struct SDL_TrayMenu SDL_TrayMenu;
typedef struct SDL_TrayEntry SDL_TrayEntry;
typedef struct SDL_Renderer SDL_Renderer;
typedef struct SDL_Texture SDL_Texture;
typedef struct SDL_Cursor SDL_Cursor;
typedef struct TTF_Font TTF_Font;
typedef struct { float x,y,w,h; } SDL_FRect;
typedef struct { int32_t x,y,w,h; } SDL_Rect;
typedef struct { uint8_t r,g,b,a; } ZSDL_Color;

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
typedef bool (*PFN_SDL_StartTextInput)(SDL_Window*);
typedef bool (*PFN_SDL_StopTextInput)(SDL_Window*);
typedef SDL_Cursor *(*PFN_SDL_CreateSystemCursor)(int);
typedef bool (*PFN_SDL_SetCursor)(SDL_Cursor*);
typedef void (*PFN_SDL_DestroyCursor)(SDL_Cursor*);
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
typedef SDL_Renderer *(*PFN_SDL_CreateRenderer)(SDL_Window*,const char*);
typedef void (*PFN_SDL_DestroyRenderer)(SDL_Renderer*);
typedef bool (*PFN_SDL_SetRenderDrawColor)(SDL_Renderer*,uint8_t,uint8_t,uint8_t,uint8_t);
typedef bool (*PFN_SDL_SetRenderDrawBlendMode)(SDL_Renderer*,uint32_t);
typedef bool (*PFN_SDL_SetRenderClipRect)(SDL_Renderer*,const SDL_Rect*);
typedef bool (*PFN_SDL_RenderClear)(SDL_Renderer*);
typedef bool (*PFN_SDL_RenderFillRect)(SDL_Renderer*,const SDL_FRect*);
typedef bool (*PFN_SDL_RenderRect)(SDL_Renderer*,const SDL_FRect*);
typedef bool (*PFN_SDL_RenderLine)(SDL_Renderer*,float,float,float,float);
typedef bool (*PFN_SDL_RenderPresent)(SDL_Renderer*);
typedef bool (*PFN_SDL_RenderDebugText)(SDL_Renderer*,float,float,const char*);
typedef SDL_Texture *(*PFN_SDL_CreateTextureFromSurface)(SDL_Renderer*,SDL_Surface*);
typedef SDL_Surface *(*PFN_SDL_CreateSurfaceFrom)(int,int,uint32_t,void*,int);
typedef SDL_Surface *(*PFN_SDL_RenderReadPixels)(SDL_Renderer*,const SDL_Rect*);
typedef SDL_Surface *(*PFN_SDL_ConvertSurface)(SDL_Surface*,uint32_t);
typedef bool (*PFN_SDL_RenderTexture)(SDL_Renderer*,SDL_Texture*,const SDL_FRect*,const SDL_FRect*);
typedef void (*PFN_SDL_DestroyTexture)(SDL_Texture*);
typedef bool (*PFN_SDL_GetTextureSize)(SDL_Texture*,float*,float*);

typedef bool (*PFN_TTF_Init)(void);
typedef void (*PFN_TTF_Quit)(void);
typedef TTF_Font *(*PFN_TTF_OpenFont)(const char*,float);
typedef void (*PFN_TTF_CloseFont)(TTF_Font*);
typedef void (*PFN_TTF_SetFontStyle)(TTF_Font*,uint32_t);
typedef bool (*PFN_TTF_GetStringSize)(TTF_Font*,const char*,size_t,int*,int*);
typedef bool (*PFN_TTF_GetStringSizeWrapped)(TTF_Font*,const char*,size_t,int,int*,int*);
typedef SDL_Surface *(*PFN_TTF_RenderText_Blended)(TTF_Font*,const char*,size_t,ZSDL_Color);
typedef SDL_Surface *(*PFN_TTF_RenderText_Blended_Wrapped)(TTF_Font*,const char*,size_t,ZSDL_Color,int);

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
static PFN_SDL_StartTextInput p_StartTextInput; static PFN_SDL_StopTextInput p_StopTextInput;
static PFN_SDL_CreateSystemCursor p_CreateSystemCursor; static PFN_SDL_SetCursor p_SetCursor; static PFN_SDL_DestroyCursor p_DestroyCursor;
static PFN_SDL_SetClipboardText p_SetClipboardText; static PFN_SDL_GetClipboardText p_GetClipboardText; static PFN_SDL_free p_free;
static PFN_SDL_IOFromFile p_IOFromFile; static PFN_SDL_LoadBMP_IO p_LoadBMP_IO; static PFN_SDL_DestroySurface p_DestroySurface; static PFN_SDL_SetWindowIcon p_SetWindowIcon;
static PFN_SDL_CreateTray p_CreateTray; static PFN_SDL_CreateTrayMenu p_CreateTrayMenu; static PFN_SDL_InsertTrayEntryAt p_InsertTrayEntryAt; static PFN_SDL_SetTrayEntryCallback p_SetTrayEntryCallback; static PFN_SDL_SetTrayTooltip p_SetTrayTooltip; static PFN_SDL_DestroyTray p_DestroyTray;
static PFN_SDL_CreateRenderer p_CreateRenderer; static PFN_SDL_DestroyRenderer p_DestroyRenderer; static PFN_SDL_SetRenderDrawColor p_SetRenderDrawColor; static PFN_SDL_SetRenderDrawBlendMode p_SetRenderDrawBlendMode; static PFN_SDL_SetRenderClipRect p_SetRenderClipRect; static PFN_SDL_RenderClear p_RenderClear; static PFN_SDL_RenderFillRect p_RenderFillRect; static PFN_SDL_RenderRect p_RenderRect; static PFN_SDL_RenderLine p_RenderLine; static PFN_SDL_RenderPresent p_RenderPresent; static PFN_SDL_RenderDebugText p_RenderDebugText; static PFN_SDL_CreateTextureFromSurface p_CreateTextureFromSurface; static PFN_SDL_CreateSurfaceFrom p_CreateSurfaceFrom; static PFN_SDL_RenderReadPixels p_RenderReadPixels; static PFN_SDL_ConvertSurface p_ConvertSurface; static PFN_SDL_RenderTexture p_RenderTexture; static PFN_SDL_DestroyTexture p_DestroyTexture; static PFN_SDL_GetTextureSize p_GetTextureSize;
static char zsdl_error[512];

static void *zttf_lib = NULL;
static bool zttf_initialized = false;
static PFN_TTF_Init p_TTF_Init; static PFN_TTF_Quit p_TTF_Quit; static PFN_TTF_OpenFont p_TTF_OpenFont; static PFN_TTF_CloseFont p_TTF_CloseFont; static PFN_TTF_SetFontStyle p_TTF_SetFontStyle; static PFN_TTF_GetStringSize p_TTF_GetStringSize; static PFN_TTF_GetStringSizeWrapped p_TTF_GetStringSizeWrapped; static PFN_TTF_RenderText_Blended p_TTF_RenderText_Blended; static PFN_TTF_RenderText_Blended_Wrapped p_TTF_RenderText_Blended_Wrapped;
typedef struct ZTTFFontCache { char path[4096]; float size; uint32_t style; TTF_Font *font; struct ZTTFFontCache *next; } ZTTFFontCache;
static ZTTFFontCache *zttf_fonts = NULL;

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
    LOAD_REQ(PollEvent); LOAD_REQ(WaitEvent); LOAD_REQ(WaitEventTimeout); LOAD_REQ(GetKeyName); LOAD_REQ(StartTextInput); LOAD_REQ(StopTextInput); LOAD_OPT(CreateSystemCursor); LOAD_OPT(SetCursor); LOAD_OPT(DestroyCursor); LOAD_REQ(SetClipboardText); LOAD_REQ(GetClipboardText); LOAD_REQ(free);
    LOAD_OPT(IOFromFile); LOAD_OPT(LoadBMP_IO); LOAD_OPT(DestroySurface); LOAD_OPT(SetWindowIcon);
    LOAD_OPT(CreateTray); LOAD_OPT(CreateTrayMenu); LOAD_OPT(InsertTrayEntryAt); LOAD_OPT(SetTrayEntryCallback); LOAD_OPT(SetTrayTooltip); LOAD_OPT(DestroyTray);
    LOAD_REQ(CreateRenderer); LOAD_REQ(DestroyRenderer); LOAD_REQ(SetRenderDrawColor); LOAD_OPT(SetRenderDrawBlendMode); LOAD_OPT(SetRenderClipRect); LOAD_REQ(RenderClear); LOAD_REQ(RenderFillRect); LOAD_REQ(RenderRect); LOAD_REQ(RenderLine); LOAD_REQ(RenderPresent); LOAD_OPT(RenderDebugText); LOAD_OPT(CreateTextureFromSurface); LOAD_OPT(CreateSurfaceFrom); LOAD_OPT(RenderReadPixels); LOAD_OPT(ConvertSurface); LOAD_OPT(RenderTexture); LOAD_OPT(DestroyTexture); LOAD_OPT(GetTextureSize);
    return true;
}

static bool zttf_file_exists(const char *path){return path&&*path&&access(path,R_OK)==0;}
static bool zttf_contains_ci(const char*text,const char*needle){if(!text||!needle||!*needle)return false;size_t n=strlen(needle);for(const char*p=text;*p;p++){size_t i=0;while(i<n&&p[i]&&tolower((unsigned char)p[i])==tolower((unsigned char)needle[i]))i++;if(i==n)return true;}return false;}
static bool zttf_bold(const char *weight){if(!weight)return false;if(strcasecmp(weight,"bold")==0||strcasecmp(weight,"semibold")==0)return true;char*end=NULL;long value=strtol(weight,&end,10);return end!=weight&&value>=600;}
static bool zttf_italic(const char *style){return style&&(strcasecmp(style,"italic")==0||strcasecmp(style,"oblique")==0);}
static const char*zttf_resolve_font(const char*family,const char*explicit_path,const char*weight,const char*style){
    static char selected[4096];selected[0]=0;
    const char*env=getenv("ZUMBRA_UI_FONT");
    if(zttf_file_exists(explicit_path)){snprintf(selected,sizeof(selected),"%s",explicit_path);return selected;}
    if(family&&strchr(family,'/')&&zttf_file_exists(family)){snprintf(selected,sizeof(selected),"%s",family);return selected;}
    if(zttf_file_exists(env)){snprintf(selected,sizeof(selected),"%s",env);return selected;}
    bool mono=family&&(zttf_contains_ci(family,"mono")||zttf_contains_ci(family,"code"));
    bool serif=family&&zttf_contains_ci(family,"serif")&&!mono;
    bool bold=zttf_bold(weight),italic=zttf_italic(style);
    const char*candidates[24];size_t n=0;
    if(mono){
        if(bold&&italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono-BoldOblique.ttf";
        else if(bold)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf";
        else if(italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Oblique.ttf";
        candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf";
        candidates[n++]="/usr/share/fonts/truetype/liberation2/LiberationMono-Regular.ttf";
    } else if(serif){
        if(bold&&italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif-BoldItalic.ttf";
        else if(bold)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif-Bold.ttf";
        else if(italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif-Italic.ttf";
        candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf";
        candidates[n++]="/usr/share/fonts/truetype/liberation2/LiberationSerif-Regular.ttf";
    } else {
        if(bold&&italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans-BoldOblique.ttf";
        else if(bold)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf";
        else if(italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans-Oblique.ttf";
        candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf";
        candidates[n++]="/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf";
        candidates[n++]="/usr/share/fonts/opentype/noto/NotoSans-Regular.ttf";
    }
    candidates[n]=NULL;for(size_t i=0;i<n;i++)if(zttf_file_exists(candidates[i])){snprintf(selected,sizeof(selected),"%s",candidates[i]);return selected;}return NULL;
}
static bool zttf_load(void){
    if(zttf_initialized)return true;if(zttf_lib==NULL){const char*names[]={"libSDL3_ttf.so.0","libSDL3_ttf.so",NULL};for(int i=0;names[i];i++){zttf_lib=dlopen(names[i],RTLD_NOW|RTLD_LOCAL);if(zttf_lib)break;}}
    if(!zttf_lib)return false;
#define ZTTF_REQ(name) do{p_TTF_##name=(PFN_TTF_##name)dlsym(zttf_lib,"TTF_" #name);if(!p_TTF_##name)return false;}while(0)
#define ZTTF_OPT(name) do{p_TTF_##name=(PFN_TTF_##name)dlsym(zttf_lib,"TTF_" #name);}while(0)
    ZTTF_REQ(Init);ZTTF_REQ(Quit);ZTTF_REQ(OpenFont);ZTTF_REQ(CloseFont);ZTTF_OPT(SetFontStyle);ZTTF_REQ(GetStringSize);ZTTF_OPT(GetStringSizeWrapped);ZTTF_REQ(RenderText_Blended);ZTTF_OPT(RenderText_Blended_Wrapped);
#undef ZTTF_REQ
#undef ZTTF_OPT
    if(!p_TTF_Init())return false;zttf_initialized=true;return true;
}
static uint32_t zttf_style_flags(const char*weight,const char*style){uint32_t flags=0;if(zttf_bold(weight))flags|=0x01u;if(zttf_italic(style))flags|=0x02u;return flags;}
static TTF_Font*zttf_font(const char*family,const char*path,float size,const char*weight,const char*style){if(size<1)size=14;if(!zttf_load())return NULL;const char*resolved=zttf_resolve_font(family,path,weight,style);if(!resolved)return NULL;uint32_t flags=zttf_style_flags(weight,style);for(ZTTFFontCache*c=zttf_fonts;c;c=c->next)if(c->size==size&&c->style==flags&&strcmp(c->path,resolved)==0)return c->font;TTF_Font*font=p_TTF_OpenFont(resolved,size);if(!font)return NULL;if(p_TTF_SetFontStyle)p_TTF_SetFontStyle(font,flags);ZTTFFontCache*c=(ZTTFFontCache*)calloc(1,sizeof(*c));if(!c){p_TTF_CloseFont(font);return NULL;}snprintf(c->path,sizeof(c->path),"%s",resolved);c->size=size;c->style=flags;c->font=font;c->next=zttf_fonts;zttf_fonts=c;return font;}
static size_t zttf_codepoints(const char*text){size_t count=0;if(!text)return 0;for(const unsigned char*p=(const unsigned char*)text;*p;p++)if((*p&0xC0)!=0x80)count++;return count;}
static bool zsdl_measure_text(const char*text,const char*family,const char*path,float size,const char*weight,const char*style,float wrap,float*out_w,float*out_h){
    if(!text)text="";TTF_Font*font=zttf_font(family,path,size,weight,style);int w=0,h=0;bool ok=false;if(font){if(wrap>0&&p_TTF_GetStringSizeWrapped)ok=p_TTF_GetStringSizeWrapped(font,text,0,(int)wrap,&w,&h);else ok=p_TTF_GetStringSize(font,text,0,&w,&h);}if(!ok){size_t count=zttf_codepoints(text);w=(int)((double)count*(size>0?size:14)*0.58);h=(int)((size>0?size:14)*1.25);if(wrap>0&&w>(int)wrap){int lines=(w+(int)wrap-1)/(int)wrap;w=(int)wrap;h*=lines;}}if(out_w)*out_w=(float)w;if(out_h)*out_h=(float)h;return font!=NULL&&ok;
}
static bool zsdl_text_ex(SDL_Renderer*r,float x,float y,const char*text,uint8_t red,uint8_t green,uint8_t blue,uint8_t alpha,const char*family,const char*path,float size,const char*weight,const char*style,float wrap){
    if(!text||!*text)return true;TTF_Font*font=zttf_font(family,path,size,weight,style);if(font&&p_CreateTextureFromSurface&&p_RenderTexture&&p_DestroyTexture&&p_DestroySurface){ZSDL_Color color={red,green,blue,alpha};SDL_Surface*surface=(wrap>0&&p_TTF_RenderText_Blended_Wrapped)?p_TTF_RenderText_Blended_Wrapped(font,text,0,color,(int)wrap):p_TTF_RenderText_Blended(font,text,0,color);if(surface){SDL_Texture*texture=p_CreateTextureFromSurface(r,surface);p_DestroySurface(surface);if(texture){float w=0,h=0;if(p_GetTextureSize)p_GetTextureSize(texture,&w,&h);if(w<=0||h<=0)zsdl_measure_text(text,family,path,size,weight,style,wrap,&w,&h);SDL_FRect dst={x,y,w,h};bool ok=p_RenderTexture(r,texture,NULL,&dst);p_DestroyTexture(texture);return ok;}}}
    if(p_RenderDebugText){p_SetRenderDrawColor(r,red,green,blue,alpha);return p_RenderDebugText(r,x,y,text);}return false;
}
static bool zsdl_ttf_available(void){return zttf_load();}
static void zttf_shutdown(void){ZTTFFontCache*c=zttf_fonts;while(c){ZTTFFontCache*n=c->next;if(c->font&&p_TTF_CloseFont)p_TTF_CloseFont(c->font);free(c);c=n;}zttf_fonts=NULL;if(zttf_initialized&&p_TTF_Quit)p_TTF_Quit();zttf_initialized=false;if(zttf_lib){dlclose(zttf_lib);zttf_lib=NULL;}}
static const char *zsdl_last_error(void){ if(zsdl_error[0])return zsdl_error; if(p_GetError){const char*e=p_GetError();if(e&&*e)return e;} return "unknown SDL3 error"; }
static bool zsdl_init(const char*name,const char*version,const char*identifier){ if(!zsdl_load())return false; if(p_SetAppMetadata)p_SetAppMetadata(name,version,identifier); if(!p_Init(0x00000020u)){snprintf(zsdl_error,sizeof(zsdl_error),"%s",p_GetError());return false;} return true; }
static SDL_Cursor *zsdl_system_cursors[3]={NULL,NULL,NULL};
static bool zsdl_set_system_cursor(int kind){
    if(!p_CreateSystemCursor||!p_SetCursor)return false;
    if(kind<0||kind>2)kind=0;
    int system_kind=kind==1?11:(kind==2?1:0);
    if(!zsdl_system_cursors[kind])zsdl_system_cursors[kind]=p_CreateSystemCursor(system_kind);
    return zsdl_system_cursors[kind]&&p_SetCursor(zsdl_system_cursors[kind]);
}
static void zsdl_destroy_system_cursors(void){if(!p_DestroyCursor)return;for(int i=0;i<3;i++){if(zsdl_system_cursors[i]){p_DestroyCursor(zsdl_system_cursors[i]);zsdl_system_cursors[i]=NULL;}}}
static void zsdl_quit(void){zttf_shutdown();zsdl_destroy_system_cursors();if(p_Quit)p_Quit();}
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
static bool zsdl_start_text_input(SDL_Window*w){return w&&p_StartTextInput&&p_StartTextInput(w);}
static bool zsdl_stop_text_input(SDL_Window*w){return w&&p_StopTextInput&&p_StopTextInput(w);}
static bool zsdl_set_icon(SDL_Window*w,const char*path){if(!p_IOFromFile||!p_LoadBMP_IO||!p_SetWindowIcon||!p_DestroySurface){snprintf(zsdl_error,sizeof(zsdl_error),"SDL3 BMP icon APIs are unavailable");return false;}SDL_IOStream*io=p_IOFromFile(path,"rb");if(!io)return false;SDL_Surface*s=p_LoadBMP_IO(io,true);if(!s)return false;bool ok=p_SetWindowIcon(w,s);p_DestroySurface(s);return ok;}
static SDL_Renderer*zsdl_renderer(SDL_Window*w){SDL_Renderer*r=p_CreateRenderer(w,NULL);if(r&&p_SetRenderDrawBlendMode)p_SetRenderDrawBlendMode(r,1u);return r;} static void zsdl_destroy_renderer(SDL_Renderer*r){if(r)p_DestroyRenderer(r);}
static bool zsdl_color(SDL_Renderer*r,uint8_t red,uint8_t green,uint8_t blue,uint8_t alpha){return p_SetRenderDrawColor(r,red,green,blue,alpha);} static bool zsdl_clear(SDL_Renderer*r){return p_RenderClear(r);}
static bool zsdl_clip(SDL_Renderer*r,int32_t x,int32_t y,int32_t w,int32_t h){if(!p_SetRenderClipRect)return true;SDL_Rect q={x,y,w,h};return p_SetRenderClipRect(r,&q);} static bool zsdl_clip_reset(SDL_Renderer*r){return !p_SetRenderClipRect||p_SetRenderClipRect(r,NULL);}
static bool zsdl_fill(SDL_Renderer*r,float x,float y,float w,float h){SDL_FRect q={x,y,w,h};return p_RenderFillRect(r,&q);} static bool zsdl_stroke(SDL_Renderer*r,float x,float y,float w,float h){SDL_FRect q={x,y,w,h};return p_RenderRect(r,&q);} static bool zsdl_line(SDL_Renderer*r,float x1,float y1,float x2,float y2){return p_RenderLine(r,x1,y1,x2,y2);} static bool zsdl_present(SDL_Renderer*r){return p_RenderPresent(r);}
typedef SDL_Surface *(*PFN_IMG_Load)(const char*);
static void *zimg_lib=NULL;static PFN_IMG_Load p_IMG_Load=NULL;
typedef void GdkPixbuf;typedef void GError;
typedef GdkPixbuf *(*PFN_GDK_PIXBUF_NEW_FROM_FILE)(const char*,GError**);typedef int(*PFN_GDK_PIXBUF_GET_INT)(const GdkPixbuf*);typedef unsigned char*(*PFN_GDK_PIXBUF_GET_PIXELS)(const GdkPixbuf*);typedef void(*PFN_G_OBJECT_UNREF)(void*);
static void*zpix_lib=NULL,*zgobject_lib=NULL;static PFN_GDK_PIXBUF_NEW_FROM_FILE p_gdk_new=NULL;static PFN_GDK_PIXBUF_GET_INT p_gdk_width=NULL,p_gdk_height=NULL,p_gdk_stride=NULL,p_gdk_channels=NULL,p_gdk_alpha=NULL;static PFN_GDK_PIXBUF_GET_PIXELS p_gdk_pixels=NULL;static PFN_G_OBJECT_UNREF p_g_object_unref=NULL;
#define ZSDL_PIXELFORMAT_RGBA32 0x16762004u
typedef struct{uint32_t flags,format;int w,h,pitch;void*pixels;int refcount;void*reserved;}ZSDLSurfaceView;
static void zsdl_image_loaders(void){
    if(!zimg_lib){const char*names[]={"libSDL3_image.so.0","libSDL3_image.so",NULL};for(int i=0;names[i];i++){zimg_lib=dlopen(names[i],RTLD_NOW|RTLD_LOCAL);if(zimg_lib)break;}if(zimg_lib)p_IMG_Load=(PFN_IMG_Load)dlsym(zimg_lib,"IMG_Load");}
    if(!zpix_lib){zpix_lib=dlopen("libgdk_pixbuf-2.0.so.0",RTLD_NOW|RTLD_LOCAL);zgobject_lib=dlopen("libgobject-2.0.so.0",RTLD_NOW|RTLD_LOCAL);if(zpix_lib&&zgobject_lib){p_gdk_new=(PFN_GDK_PIXBUF_NEW_FROM_FILE)dlsym(zpix_lib,"gdk_pixbuf_new_from_file");p_gdk_width=(PFN_GDK_PIXBUF_GET_INT)dlsym(zpix_lib,"gdk_pixbuf_get_width");p_gdk_height=(PFN_GDK_PIXBUF_GET_INT)dlsym(zpix_lib,"gdk_pixbuf_get_height");p_gdk_stride=(PFN_GDK_PIXBUF_GET_INT)dlsym(zpix_lib,"gdk_pixbuf_get_rowstride");p_gdk_channels=(PFN_GDK_PIXBUF_GET_INT)dlsym(zpix_lib,"gdk_pixbuf_get_n_channels");p_gdk_alpha=(PFN_GDK_PIXBUF_GET_INT)dlsym(zpix_lib,"gdk_pixbuf_get_has_alpha");p_gdk_pixels=(PFN_GDK_PIXBUF_GET_PIXELS)dlsym(zpix_lib,"gdk_pixbuf_get_pixels");p_g_object_unref=(PFN_G_OBJECT_UNREF)dlsym(zgobject_lib,"g_object_unref");}}
}
static SDL_Surface*zsdl_load_image_surface(const char*path,unsigned char**owned){
    *owned=NULL;zsdl_image_loaders();if(p_IMG_Load){SDL_Surface*s=p_IMG_Load(path);if(s)return s;}
    if(p_gdk_new&&p_gdk_width&&p_gdk_height&&p_gdk_stride&&p_gdk_channels&&p_gdk_pixels&&p_g_object_unref&&p_CreateSurfaceFrom){GError*err=NULL;GdkPixbuf*pix=p_gdk_new(path,&err);if(pix){int w=p_gdk_width(pix),h=p_gdk_height(pix),stride=p_gdk_stride(pix),channels=p_gdk_channels(pix),alpha=p_gdk_alpha?p_gdk_alpha(pix):channels==4;unsigned char*src=p_gdk_pixels(pix),*rgba=(unsigned char*)malloc((size_t)w*(size_t)h*4u);if(rgba){for(int y=0;y<h;y++)for(int x=0;x<w;x++){unsigned char*q=rgba+((size_t)y*(size_t)w+(size_t)x)*4u,*v=src+(size_t)y*(size_t)stride+(size_t)x*(size_t)channels;q[0]=v[0];q[1]=v[1];q[2]=v[2];q[3]=alpha&&channels>3?v[3]:255;}SDL_Surface*s=p_CreateSurfaceFrom(w,h,ZSDL_PIXELFORMAT_RGBA32,rgba,w*4);if(s){*owned=rgba;p_g_object_unref(pix);return s;}free(rgba);}p_g_object_unref(pix);}}
    if(p_IOFromFile&&p_LoadBMP_IO){SDL_IOStream*io=p_IOFromFile(path,"rb");if(io)return p_LoadBMP_IO(io,true);}return NULL;
}
static bool zsdl_image(SDL_Renderer*r,const char*path,float x,float y,float w,float h,const char*fit){if(!p_DestroySurface||!p_CreateTextureFromSurface||!p_RenderTexture||!p_DestroyTexture)return false;unsigned char*owned=NULL;SDL_Surface*s=zsdl_load_image_surface(path,&owned);if(!s)return false;SDL_Texture*t=p_CreateTextureFromSurface(r,s);p_DestroySurface(s);if(owned)free(owned);if(!t)return false;float tw=0,th=0;if(p_GetTextureSize)p_GetTextureSize(t,&tw,&th);SDL_FRect d={x,y,w,h},src={0,0,tw,th};SDL_FRect*source=NULL;if(fit&&strcmp(fit,"contain")==0&&tw>0&&th>0){float sx=w/tw,sy=h/th,scale=sx<sy?sx:sy;d.w=tw*scale;d.h=th*scale;d.x=x+(w-d.w)/2;d.y=y+(h-d.h)/2;}else if(fit&&strcmp(fit,"cover")==0&&tw>0&&th>0&&w>0&&h>0){float target=w/h,actual=tw/th;if(actual>target){src.w=th*target;src.x=(tw-src.w)/2;}else{src.h=tw/target;src.y=(th-src.h)/2;}source=&src;}bool ok=p_RenderTexture(r,t,source,&d);p_DestroyTexture(t);return ok;}
static void zsdl_box_blur_rgba(unsigned char*pixels,int w,int h,int pitch,int radius){if(!pixels||w<2||h<2||radius<1)return;size_t bytes=(size_t)pitch*(size_t)h;unsigned char*src=(unsigned char*)malloc(bytes),*tmp=(unsigned char*)malloc(bytes);if(!src||!tmp){free(src);free(tmp);return;}memcpy(src,pixels,bytes);for(int y=0;y<h;y++)for(int x=0;x<w;x++){int count=0,sum[4]={0,0,0,0};for(int k=-radius;k<=radius;k++){int sx=x+k;if(sx<0)sx=0;if(sx>=w)sx=w-1;unsigned char*q=src+(size_t)y*(size_t)pitch+(size_t)sx*4u;for(int c=0;c<4;c++)sum[c]+=q[c];count++;}unsigned char*d=tmp+(size_t)y*(size_t)pitch+(size_t)x*4u;for(int c=0;c<4;c++)d[c]=(unsigned char)(sum[c]/count);}for(int y=0;y<h;y++)for(int x=0;x<w;x++){int count=0,sum[4]={0,0,0,0};for(int k=-radius;k<=radius;k++){int sy=y+k;if(sy<0)sy=0;if(sy>=h)sy=h-1;unsigned char*q=tmp+(size_t)sy*(size_t)pitch+(size_t)x*4u;for(int c=0;c<4;c++)sum[c]+=q[c];count++;}unsigned char*d=pixels+(size_t)y*(size_t)pitch+(size_t)x*4u;for(int c=0;c<4;c++)d[c]=(unsigned char)(sum[c]/count);}free(src);free(tmp);}
static bool zsdl_blur_backdrop(SDL_Renderer*r,int w,int h,int radius){if(!p_RenderReadPixels||!p_ConvertSurface||!p_CreateTextureFromSurface||!p_RenderTexture||!p_DestroyTexture||!p_DestroySurface)return false;SDL_Surface*raw=p_RenderReadPixels(r,NULL);if(!raw)return false;SDL_Surface*rgba=p_ConvertSurface(raw,ZSDL_PIXELFORMAT_RGBA32);p_DestroySurface(raw);if(!rgba)return false;ZSDLSurfaceView*v=(ZSDLSurfaceView*)rgba;if(v->pixels&&v->w>0&&v->h>0)zsdl_box_blur_rgba((unsigned char*)v->pixels,v->w,v->h,v->pitch,radius);SDL_Texture*t=p_CreateTextureFromSurface(r,rgba);p_DestroySurface(rgba);if(!t)return false;SDL_FRect d={0,0,(float)w,(float)h};bool ok=p_RenderTexture(r,t,NULL,&d);p_DestroyTexture(t);return ok;}

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
    else if(c->type==0x403){ZSDL_MouseWheelEvent*w=(ZSDL_MouseWheelEvent*)&e;out->window_id=w->windowID;out->x=w->mouse_x;out->y=w->mouse_y;out->dx=w->x;out->dy=w->y;out->data1=w->direction;}
    else if(c->type>=0x1000&&c->type<=0x1004){ZSDL_DropEvent*d=(ZSDL_DropEvent*)&e;out->window_id=d->windowID;out->x=d->x;out->y=d->y;zsdl_copy(out->text,sizeof(out->text),d->data);zsdl_copy(out->source,sizeof(out->source),d->source);}
    return true;}
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
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
	backend  *sdlDesktopBackend
	mu       sync.RWMutex
	handle   *C.SDL_Window
	id       int64
	open     bool
	renderer *C.SDL_Renderer
	lastUI   *object.UIRenderFrame
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
		d["buttons"] = NewInteger(int64(e.data1))
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
		d["dx"] = NewFloat(float64(e.dx))
		d["dy"] = NewFloat(float64(e.dy))
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
	saveMode := optionBool(options, "save", false) || strings.EqualFold(optionString(options, "mode", "open"), "save")
	args := []string{"--file-selection"}
	if saveMode {
		args = append(args, "--save", "--confirm-overwrite")
	} else if optionBool(options, "multiple", false) {
		args = append(args, "--multiple", "--separator=\n")
	}
	if title := optionString(options, "title", ""); title != "" {
		args = append(args, "--title="+title)
	}
	defaultPath := optionString(options, "defaultPath", "")
	if defaultPath != "" {
		args = append(args, "--filename="+defaultPath)
	}
	alternateAction := "--getopenfilename"
	if saveMode {
		alternateAction = "--getsavefilename"
	}
	out, err := runDialog("zenity", args, []string{"kdialog", alternateAction, defaultPath})
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
	if w.renderer != nil {
		C.zsdl_destroy_renderer(w.renderer)
		w.renderer = nil
	}
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
func (w *sdlWindow) SetTextInput(enabled bool) error {
	if err := w.ensure(); err != nil {
		return err
	}
	var ok bool
	if enabled {
		ok = bool(C.zsdl_start_text_input(w.handle))
	} else {
		ok = bool(C.zsdl_stop_text_input(w.handle))
	}
	if !ok {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	return nil
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

func parseUIHexColor(value string) (uint8, uint8, uint8, uint8) {
	value = strings.TrimSpace(value)
	if value == "" || value == "transparent" {
		return 0, 0, 0, 0
	}
	if strings.HasPrefix(value, "#") {
		value = value[1:]
	}
	if len(value) == 3 {
		value = string([]byte{value[0], value[0], value[1], value[1], value[2], value[2]})
	}
	if len(value) == 6 {
		value += "ff"
	}
	if len(value) != 8 {
		return 0, 0, 0, 255
	}
	parsed, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return 0, 0, 0, 255
	}
	return uint8(parsed >> 24), uint8(parsed >> 16), uint8(parsed >> 8), uint8(parsed)
}
func sdlUISetColor(renderer *C.SDL_Renderer, value string) {
	r, g, b, a := parseUIHexColor(value)
	C.zsdl_color(renderer, C.uint8_t(r), C.uint8_t(g), C.uint8_t(b), C.uint8_t(a))
}
func sdlUIFill(renderer *C.SDL_Renderer, rect object.UIRect, color string) {
	_, _, _, a := parseUIHexColor(color)
	if a == 0 || rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	sdlUISetColor(renderer, color)
	C.zsdl_fill(renderer, C.float(rect.X), C.float(rect.Y), C.float(rect.Width), C.float(rect.Height))
}
func sdlUIStroke(renderer *C.SDL_Renderer, rect object.UIRect, color string) {
	_, _, _, a := parseUIHexColor(color)
	if a == 0 || rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	sdlUISetColor(renderer, color)
	C.zsdl_stroke(renderer, C.float(rect.X), C.float(rect.Y), C.float(rect.Width), C.float(rect.Height))
}
func sdlUITextStyle(props map[string]object.Object) object.UITextStyle {
	lineHeight := uiFloat(props, "lineHeight", 1.25)
	if lineHeight <= 0 {
		lineHeight = 1.25
	}
	return object.UITextStyle{
		FontFamily: uiString(props, "fontFamily", "sans"),
		FontPath:   uiString(props, "fontPath", ""),
		FontSize:   uiFloat(props, "fontSize", 14),
		FontWeight: uiString(props, "fontWeight", "normal"),
		FontStyle:  uiString(props, "fontStyle", "normal"),
		LineHeight: lineHeight,
		WrapWidth:  uiFloat(props, "wrapWidth", 0),
	}
}
func sdlMeasureUIText(text string, style object.UITextStyle) object.UITextMetrics {
	if style.FontSize <= 0 {
		style.FontSize = 14
	}
	ctext, cfamily, cpath := C.CString(text), C.CString(style.FontFamily), C.CString(style.FontPath)
	cweight, cstyle := C.CString(style.FontWeight), C.CString(style.FontStyle)
	defer C.free(unsafe.Pointer(ctext))
	defer C.free(unsafe.Pointer(cfamily))
	defer C.free(unsafe.Pointer(cpath))
	defer C.free(unsafe.Pointer(cweight))
	defer C.free(unsafe.Pointer(cstyle))
	var width, height C.float
	C.zsdl_measure_text(ctext, cfamily, cpath, C.float(style.FontSize), cweight, cstyle, C.float(style.WrapWidth), &width, &height)
	metrics := object.UITextMetrics{Width: float64(width), Height: float64(height)}
	if metrics.Height <= 0 {
		return approximateUITextMetrics(text, style)
	}
	return metrics
}
func sdlUIText(renderer *C.SDL_Renderer, x, y float64, text, color string, props map[string]object.Object) object.UITextMetrics {
	style := sdlUITextStyle(props)
	metrics := sdlMeasureUIText(text, style)
	if text == "" {
		return metrics
	}
	r, g, b, a := parseUIHexColor(color)
	ctext, cfamily, cpath := C.CString(text), C.CString(style.FontFamily), C.CString(style.FontPath)
	cweight, cstyle := C.CString(style.FontWeight), C.CString(style.FontStyle)
	defer C.free(unsafe.Pointer(ctext))
	defer C.free(unsafe.Pointer(cfamily))
	defer C.free(unsafe.Pointer(cpath))
	defer C.free(unsafe.Pointer(cweight))
	defer C.free(unsafe.Pointer(cstyle))
	C.zsdl_text_ex(renderer, C.float(x), C.float(y), ctext, C.uint8_t(r), C.uint8_t(g), C.uint8_t(b), C.uint8_t(a), cfamily, cpath, C.float(style.FontSize), cweight, cstyle, C.float(style.WrapWidth))
	return metrics
}
func sdlFitUIText(text string, maxWidth float64, props map[string]object.Object) (string, object.UITextMetrics) {
	style := sdlUITextStyle(props)
	metrics := sdlMeasureUIText(text, style)
	if maxWidth <= 0 {
		return "", object.UITextMetrics{}
	}
	if metrics.Width <= maxWidth || uiString(props, "textOverflow", "ellipsis") == "visible" {
		return text, metrics
	}
	runes := []rune(text)
	ellipsis := "…"
	for len(runes) > 0 {
		candidate := string(runes) + ellipsis
		metrics = sdlMeasureUIText(candidate, style)
		if metrics.Width <= maxWidth {
			return candidate, metrics
		}
		runes = runes[:len(runes)-1]
	}
	metrics = sdlMeasureUIText(ellipsis, style)
	if metrics.Width <= maxWidth {
		return ellipsis, metrics
	}
	return "", object.UITextMetrics{}
}
func sdlUITextCentered(renderer *C.SDL_Renderer, bounds object.UIRect, horizontalPadding float64, text, color string, props map[string]object.Object) {
	available := math.Max(0, bounds.Width-horizontalPadding*2)
	text, metrics := sdlFitUIText(text, available, props)
	y := bounds.Y + math.Max(0, (bounds.Height-metrics.Height)/2)
	x := bounds.X + horizontalPadding
	switch uiString(props, "textAlign", "left") {
	case "center":
		x = bounds.X + math.Max(horizontalPadding, (bounds.Width-metrics.Width)/2)
	case "right":
		x = bounds.X + math.Max(horizontalPadding, bounds.Width-horizontalPadding-metrics.Width)
	}
	sdlUIText(renderer, x, y, text, color, props)
}
func sdlUIItemText(item object.UIRenderItem) string {
	for _, key := range []string{"text", "label", "value", "placeholder", "title"} {
		if value, ok := item.Props[key].(*object.String); ok && value.Value != "" {
			return value.Value
		}
	}
	return ""
}

func sdlUITextSelection(props map[string]object.Object, text string) (int, int, int) {
	length := len([]rune(text))
	caret := int(uiFloat(props, "caretIndex", float64(length)))
	start := int(uiFloat(props, "selectionStart", float64(caret)))
	end := int(uiFloat(props, "selectionEnd", float64(caret)))
	if caret < 0 {
		caret = 0
	}
	if caret > length {
		caret = length
	}
	if start < 0 {
		start = 0
	}
	if start > length {
		start = length
	}
	if end < 0 {
		end = 0
	}
	if end > length {
		end = length
	}
	if start > end {
		start, end = end, start
	}
	return caret, start, end
}
func sdlUITextWindow(text string, props map[string]object.Object, available float64) ([]rune, int, int, int, int, int) {
	runes := []rune(text)
	caret, selStart, selEnd := sdlUITextSelection(props, text)
	start := 0
	for start < caret && sdlMeasureUIText(string(runes[start:caret]), sdlUITextStyle(props)).Width > available {
		start++
	}
	end := start
	for end < len(runes) {
		if sdlMeasureUIText(string(runes[start:end+1]), sdlUITextStyle(props)).Width > available {
			break
		}
		end++
	}
	return runes, start, end, caret, selStart, selEnd
}
func sdlUIRenderEditableText(renderer *C.SDL_Renderer, bounds object.UIRect, text, textColor string, props map[string]object.Object, focused bool) {
	available := math.Max(0, bounds.Width-16)
	runes, viewStart, viewEnd, caret, selStart, selEnd := sdlUITextWindow(text, props, available)
	visible := string(runes[viewStart:viewEnd])
	style := sdlUITextStyle(props)
	metrics := sdlMeasureUIText(visible, style)
	y := bounds.Y + math.Max(0, (bounds.Height-metrics.Height)/2)
	x := bounds.X + 8
	if focused && selStart != selEnd {
		visibleSelectionStart := maxInt(selStart, viewStart)
		visibleSelectionEnd := minInt(selEnd, viewEnd)
		if visibleSelectionStart < visibleSelectionEnd {
			prefixWidth := sdlMeasureUIText(string(runes[viewStart:visibleSelectionStart]), style).Width
			selectionWidth := sdlMeasureUIText(string(runes[visibleSelectionStart:visibleSelectionEnd]), style).Width
			sdlUIFill(renderer, object.UIRect{X: x + prefixWidth, Y: y - 2, Width: selectionWidth, Height: math.Max(metrics.Height+4, 18)}, uiColor(props, "selectionBackground", "#3867e855"))
		}
	}
	sdlUIText(renderer, x, y, visible, textColor, props)
	if focused {
		visibleCaret := caret
		if visibleCaret < viewStart {
			visibleCaret = viewStart
		}
		if visibleCaret > viewEnd {
			visibleCaret = viewEnd
		}
		caretWidth := sdlMeasureUIText(string(runes[viewStart:visibleCaret]), style).Width
		caretHeight := math.Min(bounds.Height-12, math.Max(14, metrics.Height))
		caretY := bounds.Y + (bounds.Height-caretHeight)/2
		sdlUISetColor(renderer, uiColor(props, "caretColor", textColor))
		C.zsdl_line(renderer, C.float(x+caretWidth), C.float(caretY), C.float(x+caretWidth), C.float(caretY+caretHeight))
	}
}
func sdlUIRenderVerticalScrollbar(renderer *C.SDL_Renderer, item object.UIRenderItem) {
	viewportHeight := item.ContentBounds.Height
	if !uiShouldRenderVerticalScrollbar(item.Props, item.ScrollContentHeight, viewportHeight) {
		return
	}
	barWidth := math.Max(4, uiFloat(item.Props, "scrollbarWidth", 8))
	gutter := math.Max(0, uiFloat(item.Props, "scrollbarGutter", 4))
	overlay := uiBool(item.Props, "scrollbarOverlay", false)
	avoidContent := uiBool(item.Props, "scrollbarAvoidContent", false)
	trackX := item.ContentBounds.X + item.ContentBounds.Width + math.Max(2, gutter)
	if overlay && !avoidContent {
		trackX = item.ContentBounds.X + item.ContentBounds.Width - barWidth - gutter
	}
	track := object.UIRect{X: trackX, Y: item.ContentBounds.Y, Width: barWidth, Height: viewportHeight}
	sdlUIFill(renderer, track, uiColor(item.Props, "scrollbarTrack", "transparent"))
	thumbHeight := math.Max(24, track.Height*viewportHeight/item.ScrollContentHeight)
	if thumbHeight > track.Height {
		thumbHeight = track.Height
	}
	maxOffset := item.ScrollContentHeight - viewportHeight
	thumbY := track.Y
	if maxOffset > 0 {
		thumbY += (track.Height - thumbHeight) * item.ScrollOffsetY / maxOffset
	}
	sdlUIFill(renderer, object.UIRect{X: track.X, Y: thumbY, Width: track.Width, Height: thumbHeight}, uiColor(item.Props, "scrollbarThumb", "#9aa7ba"))
}

func sdlUIRenderControlIcon(renderer *C.SDL_Renderer, bounds object.UIRect, icon, color string) bool {
	icon = strings.ToLower(strings.TrimSpace(icon))
	if icon == "" {
		return false
	}
	sdlUISetColor(renderer, color)
	cx, cy := bounds.X+bounds.Width/2, bounds.Y+bounds.Height/2
	size := math.Max(4, math.Min(bounds.Width, bounds.Height)*0.22)
	switch icon {
	case "close", "x":
		C.zsdl_line(renderer, C.float(cx-size), C.float(cy-size), C.float(cx+size), C.float(cy+size))
		C.zsdl_line(renderer, C.float(cx+size), C.float(cy-size), C.float(cx-size), C.float(cy+size))
	case "menu", "hamburger":
		for _, offset := range []float64{-size, 0, size} {
			C.zsdl_line(renderer, C.float(cx-size), C.float(cy+offset), C.float(cx+size), C.float(cy+offset))
		}
	case "chevron-left", "collapse-left":
		C.zsdl_line(renderer, C.float(cx+size/2), C.float(cy-size), C.float(cx-size/2), C.float(cy))
		C.zsdl_line(renderer, C.float(cx-size/2), C.float(cy), C.float(cx+size/2), C.float(cy+size))
	case "chevron-right", "expand-right":
		C.zsdl_line(renderer, C.float(cx-size/2), C.float(cy-size), C.float(cx+size/2), C.float(cy))
		C.zsdl_line(renderer, C.float(cx+size/2), C.float(cy), C.float(cx-size/2), C.float(cy+size))
	default:
		return false
	}
	return true
}

func sdlUIRenderItem(renderer *C.SDL_Renderer, item object.UIRenderItem) {
	bounds := item.Bounds
	background := uiColor(item.Props, "background", "transparent")
	border := uiColor(item.Props, "borderColor", "#cfd6e2")
	textColor := uiColor(item.Props, "textColor", "#172033")
	disabled := uiBool(item.Props, "disabled", false)
	if disabled {
		textColor = "#87909f"
	}
	switch item.Kind {
	case "container", "row", "column", "custom":
		sdlUIFill(renderer, bounds, background)
		if uiBool(item.Props, "border", false) {
			sdlUIStroke(renderer, bounds, border)
		}
	case "list", "tree":
		sdlUIFill(renderer, bounds, background)
		sdlUIStroke(renderer, bounds, border)
		items := uiObjectArray(item.Props["items"])
		rowHeight := uiFloat(item.Props, "itemHeight", 28)
		for index, value := range items {
			y := bounds.Y + 6 + float64(index)*rowHeight
			if y+8 > bounds.Y+bounds.Height {
				break
			}
			prefix := ""
			if item.Kind == "tree" {
				prefix = "- "
			}
			sdlUIText(renderer, bounds.X+8, y, prefix+uiTextDisplay(value), textColor, item.Props)
		}
	case "tabs":
		sdlUIFill(renderer, bounds, background)
		tabs := uiObjectArray(item.Props["tabs"])
		selected := int(uiFloat(item.Props, "selectedIndex", 0))
		x := bounds.X
		for index, value := range tabs {
			label := uiTextDisplay(value)
			w := math.Max(72, float64(len([]rune(label)))*8+24)
			tabBounds := object.UIRect{X: x, Y: bounds.Y, Width: w, Height: bounds.Height}
			if index == selected {
				sdlUIFill(renderer, tabBounds, uiColor(item.Props, "selectedBackground", "#ffffff"))
			}
			sdlUIStroke(renderer, tabBounds, border)
			sdlUITextCentered(renderer, tabBounds, 10, label, textColor, item.Props)
			x += w
		}
	case "menu":
		sdlUIFill(renderer, bounds, background)
		if uiBool(item.Props, "border", true) {
			sdlUIStroke(renderer, bounds, border)
		}
		if label := sdlUIItemText(item); label != "" {
			sdlUITextCentered(renderer, bounds, 10, label, textColor, item.Props)
		}
	case "table":
		sdlUIFill(renderer, bounds, background)
		sdlUIStroke(renderer, bounds, border)
		columns := uiObjectArray(item.Props["columns"])
		rows := uiObjectArray(item.Props["rows"])
		rowHeight := uiFloat(item.Props, "rowHeight", 30)
		columnCount := len(columns)
		if columnCount < 1 {
			columnCount = 1
		}
		columnWidth := bounds.Width / float64(columnCount)
		for index, column := range columns {
			x := bounds.X + float64(index)*columnWidth
			sdlUIText(renderer, x+6, bounds.Y+8, uiTextDisplay(column), textColor, item.Props)
			if index > 0 {
				sdlUISetColor(renderer, border)
				C.zsdl_line(renderer, C.float(x), C.float(bounds.Y), C.float(x), C.float(bounds.Y+bounds.Height))
			}
		}
		sdlUISetColor(renderer, border)
		C.zsdl_line(renderer, C.float(bounds.X), C.float(bounds.Y+rowHeight), C.float(bounds.X+bounds.Width), C.float(bounds.Y+rowHeight))
		for index, row := range rows {
			y := bounds.Y + rowHeight + float64(index)*rowHeight
			if y+rowHeight > bounds.Y+bounds.Height+0.5 {
				break
			}
			cells := uiObjectArray(row)
			for columnIndex := 0; columnIndex < columnCount; columnIndex++ {
				cellText := ""
				if len(cells) > columnIndex {
					cellText = uiTextDisplay(cells[columnIndex])
				} else if columnIndex == 0 {
					cellText = uiTextDisplay(row)
				}
				cellBounds := object.UIRect{X: bounds.X + float64(columnIndex)*columnWidth, Y: y, Width: columnWidth, Height: rowHeight}
				sdlUITextCentered(renderer, cellBounds, 6, cellText, textColor, item.Props)
			}
			sdlUISetColor(renderer, border)
			C.zsdl_line(renderer, C.float(bounds.X), C.float(y+rowHeight), C.float(bounds.X+bounds.Width), C.float(y+rowHeight))
		}
	case "text":
		sdlUIText(renderer, bounds.X+2, bounds.Y+4, sdlUIItemText(item), textColor, item.Props)
	case "button":
		if item.Hovered && !disabled {
			background = uiColor(item.Props, "hoverBackground", "#2f57c7")
		}
		sdlUIFill(renderer, bounds, background)
		sdlUIStroke(renderer, bounds, border)
		if !sdlUIRenderControlIcon(renderer, bounds, uiString(item.Props, "icon", ""), textColor) {
			sdlUITextCentered(renderer, bounds, 10, sdlUIItemText(item), textColor, item.Props)
		}
	case "input", "textarea", "select":
		if item.Focused {
			background = uiColor(item.Props, "focusBackground", background)
		}
		sdlUIFill(renderer, bounds, background)
		sdlUIStroke(renderer, bounds, border)
		actualText := uiString(item.Props, "value", "")
		text := actualText
		if text == "" {
			text = uiString(item.Props, "placeholder", "")
			textColor = "#87909f"
		}
		if item.Kind == "select" {
			textBounds := bounds
			textBounds.Width = math.Max(0, textBounds.Width-30)
			sdlUITextCentered(renderer, textBounds, 8, text, textColor, item.Props)
			cx := bounds.X + bounds.Width - 15
			cy := bounds.Y + bounds.Height/2
			sdlUISetColor(renderer, textColor)
			C.zsdl_line(renderer, C.float(cx-4), C.float(cy-2), C.float(cx), C.float(cy+2))
			C.zsdl_line(renderer, C.float(cx), C.float(cy+2), C.float(cx+4), C.float(cy-2))
		} else if actualText == "" && !item.Focused {
			sdlUITextCentered(renderer, bounds, 8, text, textColor, item.Props)
		} else {
			sdlUIRenderEditableText(renderer, bounds, actualText, textColor, item.Props, item.Focused)
		}
	case "checkbox", "radio":
		box := object.UIRect{X: bounds.X + 4, Y: bounds.Y + (bounds.Height-16)/2, Width: 16, Height: 16}
		sdlUIFill(renderer, box, background)
		sdlUIStroke(renderer, box, border)
		if uiBool(item.Props, "checked", false) {
			sdlUISetColor(renderer, uiColor(item.Props, "checkColor", "#3867e8"))
			C.zsdl_fill(renderer, C.float(box.X+4), C.float(box.Y+4), 8, 8)
		}
		sdlUITextCentered(renderer, object.UIRect{X: bounds.X + 20, Y: bounds.Y, Width: bounds.Width - 20, Height: bounds.Height}, 8, sdlUIItemText(item), textColor, item.Props)
	case "progress":
		sdlUIFill(renderer, bounds, background)
		value := uiClamp(uiFloat(item.Props, "value", 0), 0, uiFloat(item.Props, "max", 100))
		max := uiFloat(item.Props, "max", 100)
		if max <= 0 {
			max = 100
		}
		fill := bounds
		fill.Width = bounds.Width * value / max
		sdlUIFill(renderer, fill, uiColor(item.Props, "fill", "#3867e8"))
	case "image":
		path := uiString(item.Props, "path", "")
		if path != "" {
			c := C.CString(path)
			fit := C.CString(uiString(item.Props, "fit", "contain"))
			C.zsdl_image(renderer, c, C.float(bounds.X), C.float(bounds.Y), C.float(bounds.Width), C.float(bounds.Height), fit)
			C.free(unsafe.Pointer(fit))
			C.free(unsafe.Pointer(c))
		}
	case "modal":
		sdlUIFill(renderer, object.UIRect{X: 0, Y: 0, Width: bounds.X*2 + bounds.Width, Height: bounds.Y*2 + bounds.Height}, uiColor(item.Props, "overlay", "#00000055"))
		sdlUIFill(renderer, object.UIRect{X: bounds.X + 8, Y: bounds.Y + 10, Width: bounds.Width, Height: bounds.Height}, uiColor(item.Props, "modalShadow", "#00000040"))
		sdlUIFill(renderer, bounds, background)
		if uiBool(item.Props, "border", false) {
			sdlUIStroke(renderer, bounds, border)
		}
	case "tooltip":
		sdlUIFill(renderer, bounds, background)
		sdlUIText(renderer, bounds.X+6, bounds.Y+5, sdlUIItemText(item), textColor, item.Props)
	case "canvas":
		sdlUIFill(renderer, bounds, background)
		for _, command := range item.Commands {
			sdlUIRenderCanvasCommand(renderer, bounds, command, item.Props)
		}
	case "spacer":
		// Layout-only node.
	default:
		sdlUIFill(renderer, bounds, background)
		sdlUIText(renderer, bounds.X+4, bounds.Y+4, sdlUIItemText(item), textColor, item.Props)
	}
	if item.Focused {
		inset := math.Max(1, uiFloat(item.Props, "focusInset", 1))
		sdlUIStroke(renderer, object.UIRect{X: bounds.X + inset, Y: bounds.Y + inset, Width: math.Max(0, bounds.Width-inset*2), Height: math.Max(0, bounds.Height-inset*2)}, uiColor(item.Props, "focusColor", "#6e95ff"))
	}
}
func sdlUIRenderCanvasCommand(renderer *C.SDL_Renderer, origin object.UIRect, command object.UICanvasCommand, chartProps map[string]object.Object) {
	values := command.Values
	color := uiString(values, "color", "#172033")
	x := origin.X + uiFloat(values, "x", 0)
	y := origin.Y + uiFloat(values, "y", 0)
	switch command.Kind {
	case "rect", "fillRect":
		r := object.UIRect{X: x, Y: y, Width: uiFloat(values, "width", 10), Height: uiFloat(values, "height", 10)}
		if command.Kind == "rect" {
			sdlUIStroke(renderer, r, color)
		} else {
			sdlUIFill(renderer, r, color)
		}
	case "line":
		sdlUISetColor(renderer, color)
		C.zsdl_line(renderer, C.float(x), C.float(y), C.float(origin.X+uiFloat(values, "x2", 0)), C.float(origin.Y+uiFloat(values, "y2", 0)))
	case "text":
		sdlUIText(renderer, x, y, uiString(values, "text", ""), color, values)
	case "circle", "fillCircle":
		radius := math.Max(0, uiFloat(values, "radius", 10))
		segments := int(math.Max(24, radius*3))
		sdlUISetColor(renderer, color)
		for i := 0; i < segments; i++ {
			a1 := float64(i) * 2 * math.Pi / float64(segments)
			a2 := float64(i+1) * 2 * math.Pi / float64(segments)
			x1, y1 := x+math.Cos(a1)*radius, y+math.Sin(a1)*radius
			x2, y2 := x+math.Cos(a2)*radius, y+math.Sin(a2)*radius
			C.zsdl_line(renderer, C.float(x1), C.float(y1), C.float(x2), C.float(y2))
			if command.Kind == "fillCircle" {
				C.zsdl_line(renderer, C.float(x), C.float(y), C.float(x1), C.float(y1))
			}
		}
	case "pieChart":
		sdlUIRenderPieChart(renderer, origin, values, chartProps)
	case "barChart":
		sdlUIRenderBarChart(renderer, origin, values, chartProps)
	case "lineChart":
		sdlUIRenderLineChart(renderer, origin, values, chartProps)
	case "image":
		path := uiString(values, "path", "")
		if path != "" {
			c := C.CString(path)
			fit := C.CString(uiString(values, "fit", "contain"))
			C.zsdl_image(renderer, c, C.float(x), C.float(y), C.float(uiFloat(values, "width", 64)), C.float(uiFloat(values, "height", 64)), fit)
			C.free(unsafe.Pointer(fit))
			C.free(unsafe.Pointer(c))
		}
	}
}

func sdlUISelectPopupGeometry(item object.UIRenderItem, viewport object.UIRect) (object.UIRect, float64, int) {
	items := uiArrayStrings(item.Props["options"])
	if len(items) == 0 {
		return object.UIRect{}, 0, 0
	}
	rowHeight := math.Max(28, uiFloat(item.Props, "optionHeight", math.Max(34, item.Bounds.Height)))
	maxVisible := int(math.Max(1, uiFloat(item.Props, "maxVisibleOptions", 8)))
	visible := len(items)
	if visible > maxVisible {
		visible = maxVisible
	}
	height := rowHeight * float64(visible)
	width := math.Max(item.Bounds.Width, uiFloat(item.Props, "dropdownWidth", item.Bounds.Width))
	x := item.Bounds.X
	if x+width > viewport.X+viewport.Width {
		x = math.Max(viewport.X, viewport.X+viewport.Width-width)
	}
	y := item.Bounds.Y + item.Bounds.Height + 2
	if y+height > viewport.Y+viewport.Height && item.Bounds.Y-2-height >= viewport.Y {
		y = item.Bounds.Y - 2 - height
	}
	if y+height > viewport.Y+viewport.Height {
		height = math.Max(rowHeight, viewport.Y+viewport.Height-y)
	}
	return object.UIRect{X: x, Y: y, Width: width, Height: height}, rowHeight, visible
}

func sdlUIRenderSelectPopup(renderer *C.SDL_Renderer, item object.UIRenderItem, viewport object.UIRect) {
	if item.Kind != "select" || !uiBool(item.Props, "open", false) {
		return
	}
	options := uiArrayStrings(item.Props["options"])
	popup, rowHeight, visible := sdlUISelectPopupGeometry(item, viewport)
	if len(options) == 0 || visible == 0 {
		return
	}
	offset := int(uiFloat(item.Props, "popupOffset", 0))
	maxOffset := len(options) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	shadow := object.UIRect{X: popup.X + 4, Y: popup.Y + 5, Width: popup.Width, Height: popup.Height}
	sdlUIFill(renderer, shadow, uiColor(item.Props, "dropdownShadow", "#00000030"))
	sdlUIFill(renderer, popup, uiColor(item.Props, "dropdownBackground", uiColor(item.Props, "background", "#ffffff")))
	sdlUIStroke(renderer, popup, uiColor(item.Props, "dropdownBorderColor", uiColor(item.Props, "borderColor", "#cfd6e2")))
	selected := uiString(item.Props, "value", "")
	textColor := uiColor(item.Props, "textColor", "#172033")
	contentWidth := popup.Width
	if len(options) > visible {
		contentWidth -= 10
	}
	for row := 0; row < visible && offset+row < len(options); row++ {
		index := offset + row
		rowBounds := object.UIRect{X: popup.X + 1, Y: popup.Y + float64(row)*rowHeight + 1, Width: contentWidth - 2, Height: rowHeight - 2}
		if options[index] == selected {
			sdlUIFill(renderer, rowBounds, uiColor(item.Props, "selectedOptionBackground", "#dfe8ff"))
		}
		sdlUITextCentered(renderer, rowBounds, 10, options[index], textColor, item.Props)
		if row > 0 {
			sdlUIFill(renderer, object.UIRect{X: popup.X + 1, Y: popup.Y + float64(row)*rowHeight, Width: contentWidth - 2, Height: 1}, uiColor(item.Props, "optionSeparatorColor", uiColor(item.Props, "borderColor", "#cfd6e2")))
		}
	}
	if len(options) > visible {
		track := object.UIRect{X: popup.X + popup.Width - 8, Y: popup.Y + 3, Width: 5, Height: popup.Height - 6}
		sdlUIFill(renderer, track, uiColor(item.Props, "scrollbarTrack", "transparent"))
		thumbHeight := math.Max(18, track.Height*float64(visible)/float64(len(options)))
		thumbY := track.Y
		if maxOffset > 0 {
			thumbY += (track.Height - thumbHeight) * float64(offset) / float64(maxOffset)
		}
		sdlUIFill(renderer, object.UIRect{X: track.X, Y: thumbY, Width: track.Width, Height: thumbHeight}, uiColor(item.Props, "scrollbarThumb", "#9aa7ba"))
	}
}

func sdlUIChartNumbers(value object.Object) []float64 {
	items := uiObjectArray(value)
	result := make([]float64, 0, len(items))
	for _, item := range items {
		if number, ok := uiNumber(item); ok {
			result = append(result, number)
		}
	}
	return result
}
func sdlUIChartPalette(values map[string]object.Object) []string {
	colors := uiArrayStrings(values["colors"])
	if len(colors) > 0 {
		return colors
	}
	return []string{"#3867e8", "#e89b38", "#42a56b", "#9b59b6", "#e05260", "#2f9db0", "#8a6d3b", "#6c7a89"}
}
func sdlUIRenderPieChart(renderer *C.SDL_Renderer, origin object.UIRect, values, chartProps map[string]object.Object) {
	numbers := sdlUIChartNumbers(values["values"])
	if len(numbers) == 0 {
		return
	}
	colors := sdlUIChartPalette(values)
	labels := uiArrayStrings(values["labels"])
	x := origin.X + uiFloat(values, "x", 0)
	y := origin.Y + uiFloat(values, "y", 0)
	width := uiFloat(values, "width", origin.Width)
	height := uiFloat(values, "height", origin.Height)
	legend := uiBool(values, "legend", true)
	legendWidth := 0.0
	if legend {
		legendWidth = math.Min(170, width*0.42)
	}
	plotWidth := math.Max(0, width-legendWidth)
	radius := math.Max(0, math.Min(plotWidth, height)/2-8)
	cx, cy := x+plotWidth/2, y+height/2
	total := 0.0
	for _, number := range numbers {
		if number > 0 {
			total += number
		}
	}
	if total <= 0 || radius <= 0 {
		return
	}
	start := -math.Pi / 2
	for index, number := range numbers {
		if number <= 0 {
			continue
		}
		end := start + number/total*2*math.Pi
		sdlUISetColor(renderer, colors[index%len(colors)])
		steps := int(math.Max(2, math.Ceil((end-start)*radius*1.6)))
		for step := 0; step <= steps; step++ {
			angle := start + (end-start)*float64(step)/float64(steps)
			ex, ey := cx+math.Cos(angle)*radius, cy+math.Sin(angle)*radius
			C.zsdl_line(renderer, C.float(cx), C.float(cy), C.float(ex), C.float(ey))
		}
		start = end
	}
	if legend {
		legendX := x + plotWidth + 8
		for index, number := range numbers {
			rowY := y + 8 + float64(index)*24
			if rowY+18 > y+height {
				break
			}
			color := colors[index%len(colors)]
			sdlUIFill(renderer, object.UIRect{X: legendX, Y: rowY + 3, Width: 12, Height: 12}, color)
			label := strconv.FormatFloat(number, 'f', -1, 64)
			if index < len(labels) && labels[index] != "" {
				label = labels[index] + ": " + label
			}
			sdlUIText(renderer, legendX+18, rowY, label, uiString(values, "textColor", uiColor(chartProps, "textColor", "#172033")), values)
		}
	}
}
func sdlUIRenderBarChart(renderer *C.SDL_Renderer, origin object.UIRect, values, chartProps map[string]object.Object) {
	numbers := sdlUIChartNumbers(values["values"])
	if len(numbers) == 0 {
		return
	}
	colors := sdlUIChartPalette(values)
	labels := uiArrayStrings(values["labels"])
	x := origin.X + uiFloat(values, "x", 0)
	y := origin.Y + uiFloat(values, "y", 0)
	width := uiFloat(values, "width", origin.Width)
	height := uiFloat(values, "height", origin.Height)
	padding := math.Max(8, uiFloat(values, "padding", 18))
	labelHeight := 28.0
	plot := object.UIRect{X: x + padding, Y: y + padding, Width: math.Max(0, width-padding*2), Height: math.Max(0, height-padding*2-labelHeight)}
	maxValue := 0.0
	for _, number := range numbers {
		if number > maxValue {
			maxValue = number
		}
	}
	if maxValue <= 0 || plot.Width <= 0 || plot.Height <= 0 {
		return
	}
	gap := math.Max(4, uiFloat(values, "gap", 8))
	barWidth := math.Max(2, (plot.Width-gap*float64(len(numbers)+1))/float64(len(numbers)))
	for index, number := range numbers {
		barHeight := plot.Height * math.Max(0, number) / maxValue
		barX := plot.X + gap + float64(index)*(barWidth+gap)
		bar := object.UIRect{X: barX, Y: plot.Y + plot.Height - barHeight, Width: barWidth, Height: barHeight}
		sdlUIFill(renderer, bar, colors[index%len(colors)])
		valueLabel := strconv.FormatFloat(number, 'f', -1, 64)
		sdlUITextCentered(renderer, object.UIRect{X: barX, Y: bar.Y - 22, Width: barWidth, Height: 20}, 2, valueLabel, uiString(values, "textColor", uiColor(chartProps, "textColor", "#172033")), map[string]object.Object{"fontSize": NewFloat(11), "textAlign": NewString("center")})
		if index < len(labels) {
			sdlUITextCentered(renderer, object.UIRect{X: barX, Y: plot.Y + plot.Height + 4, Width: barWidth, Height: labelHeight}, 2, labels[index], uiString(values, "textColor", uiColor(chartProps, "textColor", "#172033")), map[string]object.Object{"fontSize": NewFloat(10), "textAlign": NewString("center"), "textOverflow": NewString("ellipsis")})
		}
	}
}
func sdlUIRenderLineChart(renderer *C.SDL_Renderer, origin object.UIRect, values, chartProps map[string]object.Object) {
	numbers := sdlUIChartNumbers(values["values"])
	if len(numbers) == 0 {
		return
	}
	labels := uiArrayStrings(values["labels"])
	x := origin.X + uiFloat(values, "x", 0)
	y := origin.Y + uiFloat(values, "y", 0)
	width := uiFloat(values, "width", origin.Width)
	height := uiFloat(values, "height", origin.Height)
	padding := math.Max(8, uiFloat(values, "padding", 18))
	labelHeight := 0.0
	if len(labels) > 0 {
		labelHeight = 30
	}
	valueHeight := 0.0
	if uiBool(values, "showValues", true) {
		valueHeight = 24
	}
	plot := object.UIRect{X: x + padding, Y: y + padding + valueHeight, Width: math.Max(0, width-padding*2), Height: math.Max(0, height-padding*2-labelHeight-valueHeight)}
	if plot.Width <= 0 || plot.Height <= 0 {
		return
	}
	minValue, maxValue := numbers[0], numbers[0]
	for _, number := range numbers[1:] {
		if number < minValue {
			minValue = number
		}
		if number > maxValue {
			maxValue = number
		}
	}
	if maxValue == minValue {
		minValue -= 0.5
		maxValue += 0.5
	}
	colors := sdlUIChartPalette(values)
	color := uiString(values, "color", colors[0])
	textColor := uiString(values, "textColor", uiColor(chartProps, "textColor", "#172033"))
	slot := plot.Width / float64(len(numbers))
	previousX, previousY := 0.0, 0.0
	for index, number := range numbers {
		pointX := plot.X + (float64(index)+0.5)*slot
		pointY := plot.Y + plot.Height - (number-minValue)/(maxValue-minValue)*plot.Height
		if index > 0 {
			sdlUISetColor(renderer, color)
			C.zsdl_line(renderer, C.float(previousX), C.float(previousY), C.float(pointX), C.float(pointY))
		}
		sdlUIFill(renderer, object.UIRect{X: pointX - 3, Y: pointY - 3, Width: 6, Height: 6}, color)
		cellWidth := math.Max(48, slot)
		if valueHeight > 0 {
			valueLabel := strconv.FormatFloat(number, 'f', -1, 64)
			sdlUITextCentered(renderer, object.UIRect{X: pointX - cellWidth/2, Y: pointY - 24, Width: cellWidth, Height: 20}, 2, valueLabel, textColor, map[string]object.Object{"fontSize": NewFloat(11), "textAlign": NewString("center"), "textOverflow": NewString("ellipsis")})
		}
		if index < len(labels) {
			sdlUITextCentered(renderer, object.UIRect{X: pointX - cellWidth/2, Y: plot.Y + plot.Height + 5, Width: cellWidth, Height: labelHeight - 4}, 2, labels[index], textColor, map[string]object.Object{"fontSize": NewFloat(10), "textAlign": NewString("center"), "textOverflow": NewString("ellipsis")})
		}
		previousX, previousY = pointX, pointY
	}
}

func uiRectsEqual(a, b *object.UIRect) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.X == b.X && a.Y == b.Y && a.Width == b.Width && a.Height == b.Height
}

func (w *sdlWindow) RenderUI(frame *object.UIRenderFrame) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open || w.handle == nil {
		return errors.New("window is closed")
	}
	if w.renderer == nil {
		w.renderer = C.zsdl_renderer(w.handle)
		if w.renderer == nil {
			return errors.New("SDL3 could not create a renderer: " + C.GoString(C.zsdl_last_error()))
		}
	}
	sdlUISetColor(w.renderer, frame.Background)
	if !bool(C.zsdl_clear(w.renderer)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	var activeClip *object.UIRect
	blurredBackdrop := false
	for _, item := range frame.Items {
		if item.Kind == "modal" && !blurredBackdrop {
			C.zsdl_clip_reset(w.renderer)
			radius := int(uiFloat(item.Props, "backdropBlur", 6))
			if radius > 0 {
				C.zsdl_blur_backdrop(w.renderer, C.int(frame.Width), C.int(frame.Height), C.int(radius))
			}
			blurredBackdrop = true
			activeClip = nil
		}
		if !uiRectsEqual(activeClip, item.Clip) {
			if item.Clip == nil {
				C.zsdl_clip_reset(w.renderer)
				activeClip = nil
			} else {
				clip := *item.Clip
				C.zsdl_clip(w.renderer, C.int32_t(math.Floor(clip.X)), C.int32_t(math.Floor(clip.Y)), C.int32_t(math.Ceil(clip.Width)), C.int32_t(math.Ceil(clip.Height)))
				copy := clip
				activeClip = &copy
			}
		}
		sdlUIRenderItem(w.renderer, item)
	}
	C.zsdl_clip_reset(w.renderer)
	for _, item := range frame.Items {
		if uiShouldRenderVerticalScrollbar(item.Props, item.ScrollContentHeight, item.ContentBounds.Height) {
			if item.Clip != nil {
				clip := *item.Clip
				C.zsdl_clip(w.renderer, C.int32_t(math.Floor(clip.X)), C.int32_t(math.Floor(clip.Y)), C.int32_t(math.Ceil(clip.Width)), C.int32_t(math.Ceil(clip.Height)))
			} else {
				C.zsdl_clip_reset(w.renderer)
			}
			sdlUIRenderVerticalScrollbar(w.renderer, item)
		}
	}
	C.zsdl_clip_reset(w.renderer)
	viewport := object.UIRect{X: 0, Y: 0, Width: frame.Width, Height: frame.Height}
	var modalBounds *object.UIRect
	for _, item := range frame.Items {
		if item.Kind == "modal" {
			bounds := item.Bounds
			modalBounds = &bounds
		}
	}
	cursorKind := 0
	for _, item := range frame.Items {
		if !item.Hovered {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(uiString(item.Props, "cursor", "default"))) {
		case "pointer", "hand":
			cursorKind = 1
		case "text", "ibeam", "i-beam":
			cursorKind = 2
		}
	}
	C.zsdl_set_system_cursor(C.int(cursorKind))
	for _, item := range frame.Items {
		if item.Kind != "select" || !uiBool(item.Props, "open", false) {
			continue
		}
		if modalBounds != nil {
			centerX, centerY := item.Bounds.X+item.Bounds.Width/2, item.Bounds.Y+item.Bounds.Height/2
			if !uiPointInRect(centerX, centerY, *modalBounds) {
				continue
			}
		}
		sdlUIRenderSelectPopup(w.renderer, item, viewport)
	}
	if !bool(C.zsdl_present(w.renderer)) {
		return errors.New(C.GoString(C.zsdl_last_error()))
	}
	w.lastUI = frame
	return nil
}
func (w *sdlWindow) LastUIFrame() *object.UIRenderFrame {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastUI
}
func (w *sdlWindow) MeasureUIText(text string, style object.UITextStyle) object.UITextMetrics {
	return sdlMeasureUIText(text, style)
}
