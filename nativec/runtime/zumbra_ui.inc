#if defined(ZUMBRA_ENABLE_UI)
#include <sys/stat.h>

/* Z14 retained-mode GUI toolkit. It shares Z13's SDL3 loader and uses a
 * deterministic headless layout/render path for automated tests. */

typedef struct ZUINodeNative ZUINodeNative;
typedef struct ZUIStateNative ZUIStateNative;
typedef struct ZUIThemeNative ZUIThemeNative;
typedef struct ZUIContextNative ZUIContextNative;
typedef struct ZUIBindingNative ZUIBindingNative;
typedef struct ZUISubscriberNative ZUISubscriberNative;
typedef struct ZUIReverseBindingNative ZUIReverseBindingNative;

typedef struct { double x,y,width,height; } ZUIRectNative;
typedef struct { ZValue receiver; const char *builtin; } ZUIMethodNative;

struct ZUINodeNative {
    char *id;
    char *kind;
    ZValue props;
    ZValue children;
    ZUINodeNative *parent;
    ZUIContextNative *context;
    ZUIReverseBindingNative *bindings;
    ZUIRectNative bounds;
    ZUIRectNative content_bounds;
    double scroll_offset_y;
    double content_height;
    bool visible;
    bool enabled;
};
struct ZUIBindingNative { ZUINodeNative *node; char *property; ZUIBindingNative *next; };
struct ZUIReverseBindingNative { char *property; ZUIStateNative *state; ZUIReverseBindingNative *next; };
struct ZUISubscriberNative { ZValue handler; ZUISubscriberNative *next; };
struct ZUIStateNative { ZValue value; ZUIBindingNative *bindings; ZUISubscriberNative *subscribers; };
struct ZUIThemeNative { char *name; ZValue values; };
struct ZUIContextNative {
    ZDesktopAppNative *app;
    ZDesktopWindowNative *window;
    ZUINodeNative *root;
    ZUIThemeNative *theme;
    char *focus_id;
    char *hover_id;
    bool dirty;
    bool closed;
    struct ZUIContextNative *next;
};

static int64_t z_ui_next_id=1;
static double z_ui_prop_number(ZValue props,const char*key,double fallback);
static const char*z_ui_prop_string(ZValue props,const char*key,const char*fallback);
static double z_ui_theme_number(ZUIThemeNative*t,const char*key,double fallback);
static const char*z_ui_theme_string(ZUIThemeNative*t,const char*key,const char*fallback);
#if defined(ZUMBRA_DESKTOP_SDL3)
typedef struct TTF_Font TTF_Font;
typedef struct { uint8_t r,g,b,a; } ZUITTFColor;
typedef bool (*ZUI_TTF_Init)(void);
typedef void (*ZUI_TTF_Quit)(void);
typedef TTF_Font *(*ZUI_TTF_OpenFont)(const char*,float);
typedef void (*ZUI_TTF_CloseFont)(TTF_Font*);
typedef void (*ZUI_TTF_SetFontStyle)(TTF_Font*,uint32_t);
typedef bool (*ZUI_TTF_GetStringSize)(TTF_Font*,const char*,size_t,int*,int*);
typedef bool (*ZUI_TTF_GetStringSizeWrapped)(TTF_Font*,const char*,size_t,int,int*,int*);
typedef SDL_Surface *(*ZUI_TTF_RenderText_Blended)(TTF_Font*,const char*,size_t,ZUITTFColor);
typedef SDL_Surface *(*ZUI_TTF_RenderText_Blended_Wrapped)(TTF_Font*,const char*,size_t,ZUITTFColor,int);
typedef struct ZUIFontCacheNative { char path[PATH_MAX]; float size; uint32_t style; TTF_Font *font; struct ZUIFontCacheNative *next; } ZUIFontCacheNative;
typedef struct {
    void *lib; bool initialized;
    ZUI_TTF_Init Init; ZUI_TTF_Quit Quit; ZUI_TTF_OpenFont OpenFont; ZUI_TTF_CloseFont CloseFont;
    ZUI_TTF_SetFontStyle SetFontStyle; ZUI_TTF_GetStringSize GetStringSize; ZUI_TTF_GetStringSizeWrapped GetStringSizeWrapped;
    ZUI_TTF_RenderText_Blended RenderTextBlended; ZUI_TTF_RenderText_Blended_Wrapped RenderTextBlendedWrapped;
    ZUIFontCacheNative *fonts;
} ZUITTFRuntime;
typedef struct ZUITextTextureCacheNative {
    SDL_Renderer *renderer;
    TTF_Font *font;
    char *text;
    uint32_t rgba;
    int wrap;
    SDL_Texture *texture;
    float width;
    float height;
    uint64_t used;
    struct ZUITextTextureCacheNative *next;
} ZUITextTextureCacheNative;
typedef struct ZUIImageTextureCacheNative {
    SDL_Renderer *renderer;
    char *path;
    int64_t modified_ns;
    int64_t file_size;
    SDL_Texture *texture;
    float width;
    float height;
    uint64_t used;
    struct ZUIImageTextureCacheNative *next;
} ZUIImageTextureCacheNative;
typedef struct ZUIBackdropCacheNative {
    SDL_Renderer *renderer;
    const void *owner;
    int width;
    int height;
    int radius;
    SDL_Texture *texture;
    struct ZUIBackdropCacheNative *next;
} ZUIBackdropCacheNative;
static ZUITTFRuntime z_ui_ttf;
static ZUITextTextureCacheNative *z_ui_textures=NULL;
static ZUIImageTextureCacheNative *z_ui_images=NULL;
static ZUIBackdropCacheNative *z_ui_backdrops=NULL;
static size_t z_ui_textures_count=0;
static size_t z_ui_images_count=0;
static uint64_t z_ui_cache_clock=1;
static bool z_ui_contains_ci(const char*text,const char*needle){if(text==NULL||needle==NULL||*needle=='\0')return false;size_t n=strlen(needle);for(const char*p=text;*p;p++){size_t i=0;while(i<n&&p[i]&&tolower((unsigned char)p[i])==tolower((unsigned char)needle[i]))i++;if(i==n)return true;}return false;}
static bool z_ui_file_exists(const char*path){return path!=NULL&&*path!='\0'&&z_desktop_access(path,4)==0;}
static bool z_ui_font_bold(const char*weight){if(weight==NULL)return false;if(strcasecmp(weight,"bold")==0||strcasecmp(weight,"semibold")==0)return true;char*end=NULL;long v=strtol(weight,&end,10);return end!=weight&&v>=600;}
static bool z_ui_font_italic(const char*style){return style!=NULL&&(strcasecmp(style,"italic")==0||strcasecmp(style,"oblique")==0);}
static const char*z_ui_resolve_font(const char*family,const char*explicit_path,const char*weight,const char*style){
    static char selected[PATH_MAX];selected[0]='\0';const char*env=getenv("ZUMBRA_UI_FONT");
    if(z_ui_file_exists(explicit_path)){snprintf(selected,sizeof(selected),"%s",explicit_path);return selected;}
    if(family&&strchr(family,'/')&&z_ui_file_exists(family)){snprintf(selected,sizeof(selected),"%s",family);return selected;}
    if(z_ui_file_exists(env)){snprintf(selected,sizeof(selected),"%s",env);return selected;}
    bool mono=family&&(z_ui_contains_ci(family,"mono")||z_ui_contains_ci(family,"code"));bool serif=family&&z_ui_contains_ci(family,"serif")&&!mono;bool bold=z_ui_font_bold(weight),italic=z_ui_font_italic(style);
    const char*candidates[32];size_t n=0;
#if defined(_WIN32)
    static char win_fonts[PATH_MAX];const char*windir=getenv("WINDIR");if(windir==NULL)windir="C:/Windows";snprintf(win_fonts,sizeof(win_fonts),"%s/Fonts",windir);
    static char win_paths[8][PATH_MAX];const char*files[8]={mono?"consola.ttf":serif?"times.ttf":"segoeui.ttf",mono?"cour.ttf":serif?"timesbd.ttf":"segoeuib.ttf","arial.ttf","arialbd.ttf","calibri.ttf","calibrib.ttf",NULL,NULL};
    for(size_t i=0;i<8&&files[i];i++){snprintf(win_paths[i],sizeof(win_paths[i]),"%s/%s",win_fonts,files[i]);candidates[n++]=win_paths[i];}
#elif defined(__APPLE__)
    if(mono){candidates[n++]="/System/Library/Fonts/SFNSMono.ttf";candidates[n++]="/System/Library/Fonts/Monaco.ttf";}else if(serif){candidates[n++]="/System/Library/Fonts/NewYork.ttf";candidates[n++]="/System/Library/Fonts/Supplemental/Times New Roman.ttf";}else{candidates[n++]="/System/Library/Fonts/SFNS.ttf";candidates[n++]="/System/Library/Fonts/Supplemental/Arial.ttf";}
#else
    if(mono){if(bold&&italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono-BoldOblique.ttf";else if(bold)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf";else if(italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Oblique.ttf";candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf";candidates[n++]="/usr/share/fonts/truetype/liberation2/LiberationMono-Regular.ttf";}
    else if(serif){if(bold&&italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif-BoldItalic.ttf";else if(bold)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif-Bold.ttf";else if(italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif-Italic.ttf";candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf";candidates[n++]="/usr/share/fonts/truetype/liberation2/LiberationSerif-Regular.ttf";}
    else {if(bold&&italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans-BoldOblique.ttf";else if(bold)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf";else if(italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans-Oblique.ttf";candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf";candidates[n++]="/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf";candidates[n++]="/usr/share/fonts/opentype/noto/NotoSans-Regular.ttf";}
#endif
    for(size_t i=0;i<n;i++) {
        if(z_ui_file_exists(candidates[i])) {
            snprintf(selected,sizeof(selected),"%s",candidates[i]);
            return selected;
        }
    }
    return NULL;
}
static bool z_ui_ttf_load(void){
    if(z_ui_ttf.initialized) return true;
    if(z_ui_ttf.lib==NULL) {
#if defined(_WIN32)
        const char*names[]={"SDL3_ttf.dll","libSDL3_ttf.dll",NULL};
#elif defined(__APPLE__)
        const char*names[]={"@executable_path/../Frameworks/libSDL3_ttf.0.dylib","@executable_path/../Frameworks/libSDL3_ttf.dylib","libSDL3_ttf.0.dylib","libSDL3_ttf.dylib",NULL};
#else
        const char*names[]={"libSDL3_ttf.so.0","libSDL3_ttf.so",NULL};
#endif
        for(size_t i=0;names[i];i++) {
            z_ui_ttf.lib=z_desktop_library_open(names[i]);
            if(z_ui_ttf.lib) break;
        }
    }
    if(z_ui_ttf.lib==NULL) return false;
#define ZUI_TTF_REQ(field,symbol,type) do{z_ui_ttf.field=(type)z_desktop_library_symbol(z_ui_ttf.lib,symbol);if(z_ui_ttf.field==NULL)return false;}while(0)
#define ZUI_TTF_OPT(field,symbol,type) do{z_ui_ttf.field=(type)z_desktop_library_symbol(z_ui_ttf.lib,symbol);}while(0)
    ZUI_TTF_REQ(Init,"TTF_Init",ZUI_TTF_Init);ZUI_TTF_REQ(Quit,"TTF_Quit",ZUI_TTF_Quit);ZUI_TTF_REQ(OpenFont,"TTF_OpenFont",ZUI_TTF_OpenFont);ZUI_TTF_REQ(CloseFont,"TTF_CloseFont",ZUI_TTF_CloseFont);ZUI_TTF_OPT(SetFontStyle,"TTF_SetFontStyle",ZUI_TTF_SetFontStyle);ZUI_TTF_REQ(GetStringSize,"TTF_GetStringSize",ZUI_TTF_GetStringSize);ZUI_TTF_OPT(GetStringSizeWrapped,"TTF_GetStringSizeWrapped",ZUI_TTF_GetStringSizeWrapped);ZUI_TTF_REQ(RenderTextBlended,"TTF_RenderText_Blended",ZUI_TTF_RenderText_Blended);ZUI_TTF_OPT(RenderTextBlendedWrapped,"TTF_RenderText_Blended_Wrapped",ZUI_TTF_RenderText_Blended_Wrapped);
#undef ZUI_TTF_REQ
#undef ZUI_TTF_OPT
    if(!z_ui_ttf.Init()) return false;
    z_ui_ttf.initialized=true;
    return true;
}
static uint32_t z_ui_font_flags(const char*weight,const char*style){uint32_t flags=0;if(z_ui_font_bold(weight))flags|=0x01u;if(z_ui_font_italic(style))flags|=0x02u;return flags;}
static TTF_Font*z_ui_font(const char*family,const char*path,double size,const char*weight,const char*style){if(size<1)size=14;if(!z_ui_ttf_load())return NULL;const char*resolved=z_ui_resolve_font(family,path,weight,style);if(resolved==NULL)return NULL;uint32_t flags=z_ui_font_flags(weight,style);for(ZUIFontCacheNative*c=z_ui_ttf.fonts;c;c=c->next)if(c->size==(float)size&&c->style==flags&&strcmp(c->path,resolved)==0)return c->font;TTF_Font*font=z_ui_ttf.OpenFont(resolved,(float)size);if(font==NULL)return NULL;if(z_ui_ttf.SetFontStyle)z_ui_ttf.SetFontStyle(font,flags);ZUIFontCacheNative*c=(ZUIFontCacheNative*)calloc(1,sizeof(*c));if(c==NULL){z_ui_ttf.CloseFont(font);return NULL;}snprintf(c->path,sizeof(c->path),"%s",resolved);c->size=(float)size;c->style=flags;c->font=font;c->next=z_ui_ttf.fonts;z_ui_ttf.fonts=c;return font;}
static size_t z_ui_utf8_count(const char*text){size_t count=0;if(text==NULL)return 0;for(const unsigned char*p=(const unsigned char*)text;*p;p++)if((*p&0xC0u)!=0x80u)count++;return count;}
static void z_ui_measure_text_native(const char*text,ZValue props,ZUIThemeNative*t,double wrap,double*out_w,double*out_h){if(text==NULL)text="";double size=z_ui_prop_number(props,"fontSize",z_ui_theme_number(t,"fontSize",14));const char*family=z_ui_prop_string(props,"fontFamily",z_ui_theme_string(t,"fontFamily","sans"));const char*path=z_ui_prop_string(props,"fontPath",z_ui_theme_string(t,"fontPath",""));const char*weight=z_ui_prop_string(props,"fontWeight",z_ui_theme_string(t,"fontWeight","normal"));const char*style=z_ui_prop_string(props,"fontStyle",z_ui_theme_string(t,"fontStyle","normal"));TTF_Font*font=z_ui_font(family,path,size,weight,style);int w=0,h=0;bool ok=false;if(font){if(wrap>0&&z_ui_ttf.GetStringSizeWrapped)ok=z_ui_ttf.GetStringSizeWrapped(font,text,0,(int)wrap,&w,&h);else ok=z_ui_ttf.GetStringSize(font,text,0,&w,&h);}if(!ok){w=(int)((double)z_ui_utf8_count(text)*size*0.58);h=(int)(size*z_ui_prop_number(props,"lineHeight",z_ui_theme_number(t,"lineHeight",1.25)));if(wrap>0&&w>(int)wrap){int lines=(w+(int)wrap-1)/(int)wrap;w=(int)wrap;h*=lines;}}if(out_w)*out_w=w;if(out_h)*out_h=h;}

static uint32_t z_ui_rgba_key(uint8_t red,uint8_t green,uint8_t blue,uint8_t alpha){return((uint32_t)red<<24)|((uint32_t)green<<16)|((uint32_t)blue<<8)|(uint32_t)alpha;}
static void z_ui_destroy_texture(SDL_Texture*texture){if(texture&&z_desktop_sdl.initialized&&z_desktop_sdl.DestroyTexture)z_desktop_sdl.DestroyTexture(texture);}
static void z_ui_text_cache_remove(ZUITextTextureCacheNative*target){ZUITextTextureCacheNative**link=&z_ui_textures;while(*link){if(*link==target){*link=target->next;z_ui_destroy_texture(target->texture);free(target->text);free(target);if(z_ui_textures_count>0)z_ui_textures_count--;return;}link=&(*link)->next;}}
static void z_ui_image_cache_remove(ZUIImageTextureCacheNative*target){ZUIImageTextureCacheNative**link=&z_ui_images;while(*link){if(*link==target){*link=target->next;z_ui_destroy_texture(target->texture);free(target->path);free(target);if(z_ui_images_count>0)z_ui_images_count--;return;}link=&(*link)->next;}}
static void z_ui_text_cache_trim(void){while(z_ui_textures_count>512){ZUITextTextureCacheNative*oldest=NULL;for(ZUITextTextureCacheNative*c=z_ui_textures;c;c=c->next)if(oldest==NULL||c->used<oldest->used)oldest=c;if(oldest==NULL)break;z_ui_text_cache_remove(oldest);}}
static void z_ui_image_cache_trim(void){while(z_ui_images_count>128){ZUIImageTextureCacheNative*oldest=NULL;for(ZUIImageTextureCacheNative*c=z_ui_images;c;c=c->next)if(oldest==NULL||c->used<oldest->used)oldest=c;if(oldest==NULL)break;z_ui_image_cache_remove(oldest);}}
static ZUITextTextureCacheNative*z_ui_text_cache_find(SDL_Renderer*renderer,TTF_Font*font,const char*text,uint32_t rgba,int wrap){for(ZUITextTextureCacheNative*c=z_ui_textures;c;c=c->next){if(c->renderer==renderer&&c->font==font&&c->rgba==rgba&&c->wrap==wrap&&strcmp(c->text,text)==0){c->used=z_ui_cache_clock++;return c;}}return NULL;}
static ZUITextTextureCacheNative*z_ui_text_cache_store(SDL_Renderer*renderer,TTF_Font*font,const char*text,uint32_t rgba,int wrap,SDL_Texture*texture,float width,float height){ZUITextTextureCacheNative*c=(ZUITextTextureCacheNative*)calloc(1,sizeof(*c));if(c==NULL)return NULL;c->renderer=renderer;c->font=font;c->text=strdup(text);if(c->text==NULL){free(c);return NULL;}c->rgba=rgba;c->wrap=wrap;c->texture=texture;c->width=width;c->height=height;c->used=z_ui_cache_clock++;c->next=z_ui_textures;z_ui_textures=c;z_ui_textures_count++;z_ui_text_cache_trim();return c;}
static void z_ui_file_signature(const char*path,int64_t*modified,int64_t*size){*modified=0;*size=0;if(path==NULL||*path=='\0')return;struct stat info;if(stat(path,&info)==0){*modified=(int64_t)info.st_mtime;*size=(int64_t)info.st_size;}}
static ZUIImageTextureCacheNative*z_ui_image_cache_find(SDL_Renderer*renderer,const char*path,int64_t modified,int64_t size){for(ZUIImageTextureCacheNative*c=z_ui_images;c;c=c->next){if(c->renderer==renderer&&strcmp(c->path,path)==0){if(c->modified_ns==modified&&c->file_size==size){c->used=z_ui_cache_clock++;return c;}z_ui_image_cache_remove(c);return NULL;}}return NULL;}
static ZUIImageTextureCacheNative*z_ui_image_cache_store(SDL_Renderer*renderer,const char*path,int64_t modified,int64_t size,SDL_Texture*texture,float width,float height){ZUIImageTextureCacheNative*c=(ZUIImageTextureCacheNative*)calloc(1,sizeof(*c));if(c==NULL)return NULL;c->renderer=renderer;c->path=strdup(path);if(c->path==NULL){free(c);return NULL;}c->modified_ns=modified;c->file_size=size;c->texture=texture;c->width=width;c->height=height;c->used=z_ui_cache_clock++;c->next=z_ui_images;z_ui_images=c;z_ui_images_count++;z_ui_image_cache_trim();return c;}
static void z_ui_backdrop_cache_clear_renderer(SDL_Renderer*renderer){ZUIBackdropCacheNative**link=&z_ui_backdrops;while(*link){ZUIBackdropCacheNative*c=*link;if(c->renderer==renderer){*link=c->next;z_ui_destroy_texture(c->texture);free(c);continue;}link=&c->next;}}
static ZUIBackdropCacheNative*z_ui_backdrop_cache_find(SDL_Renderer*renderer,const void*owner,int width,int height,int radius){for(ZUIBackdropCacheNative*c=z_ui_backdrops;c;c=c->next)if(c->renderer==renderer&&c->owner==owner&&c->width==width&&c->height==height&&c->radius==radius)return c;return NULL;}
static void z_ui_cache_clear_renderer(SDL_Renderer*renderer){ZUITextTextureCacheNative*t=z_ui_textures;while(t){ZUITextTextureCacheNative*next=t->next;if(t->renderer==renderer)z_ui_text_cache_remove(t);t=next;}ZUIImageTextureCacheNative*i=z_ui_images;while(i){ZUIImageTextureCacheNative*next=i->next;if(i->renderer==renderer)z_ui_image_cache_remove(i);i=next;}z_ui_backdrop_cache_clear_renderer(renderer);}
static void z_ui_cache_shutdown(void){while(z_ui_textures)z_ui_text_cache_remove(z_ui_textures);while(z_ui_images)z_ui_image_cache_remove(z_ui_images);while(z_ui_backdrops){ZUIBackdropCacheNative*next=z_ui_backdrops->next;z_ui_destroy_texture(z_ui_backdrops->texture);free(z_ui_backdrops);z_ui_backdrops=next;}z_ui_cache_clock=1;}
static void z_ui_release_renderer_resources(void*renderer){if(renderer)z_ui_cache_clear_renderer((SDL_Renderer*)renderer);}
static void z_ui_ttf_shutdown(void){ZUIFontCacheNative*c=z_ui_ttf.fonts;while(c){ZUIFontCacheNative*n=c->next;if(c->font&&z_ui_ttf.CloseFont)z_ui_ttf.CloseFont(c->font);free(c);c=n;}z_ui_ttf.fonts=NULL;if(z_ui_ttf.initialized&&z_ui_ttf.Quit)z_ui_ttf.Quit();z_ui_ttf.initialized=false;if(z_ui_ttf.lib){z_desktop_library_close(z_ui_ttf.lib);z_ui_ttf.lib=NULL;}}
#else
static void z_ui_measure_text_native(const char*text,ZValue props,ZUIThemeNative*t,double wrap,double*out_w,double*out_h){double size=z_ui_prop_number(props,"fontSize",z_ui_theme_number(t,"fontSize",14));size_t count=text?strlen(text):0;double w=(double)count*size*0.58,h=size*1.25;if(wrap>0&&w>wrap){double lines=ceil(w/wrap);w=wrap;h*=lines;}if(out_w)*out_w=w;if(out_h)*out_h=h;}
static void z_ui_ttf_shutdown(void){}
#endif
static void z_ui_runtime_init(void){
    z_ui_next_id=1;
#if defined(ZUMBRA_DESKTOP_SDL3)
    memset(&z_ui_ttf,0,sizeof(z_ui_ttf));
    z_ui_textures=NULL;z_ui_images=NULL;z_ui_backdrops=NULL;
    z_ui_textures_count=0;z_ui_images_count=0;z_ui_cache_clock=1;
#endif
}
static void z_ui_runtime_shutdown(void){
#if defined(ZUMBRA_DESKTOP_SDL3)
    z_ui_cache_shutdown();
#endif
    z_ui_ttf_shutdown();
}
static ZValue z_ui_value(ZTag tag,ZKind kind,void*p){ZValue v=z_value(tag,kind);v.as.p=p;return v;}
static ZUINodeNative*z_ui_expect_node(ZValue v){if(v.tag!=ZV_UI_NODE||v.as.p==NULL)z_fatal("expected UINode");return(ZUINodeNative*)v.as.p;}
static ZUIStateNative*z_ui_expect_state(ZValue v){if(v.tag!=ZV_UI_STATE||v.as.p==NULL)z_fatal("expected UIState");return(ZUIStateNative*)v.as.p;}
static ZUIThemeNative*z_ui_expect_theme(ZValue v){if(v.tag!=ZV_UI_THEME||v.as.p==NULL)z_fatal("expected UITheme");return(ZUIThemeNative*)v.as.p;}
static ZUIContextNative*z_ui_expect_context(ZValue v){if(v.tag!=ZV_UI_CONTEXT||v.as.p==NULL)z_fatal("expected UIContext");return(ZUIContextNative*)v.as.p;}
static double z_ui_number(ZValue v,double fallback){if(v.tag==ZV_FLOAT)return v.as.f;if(v.tag==ZV_INT)return(double)v.as.i;if(v.tag==ZV_UINT)return(double)v.as.u;return fallback;}
static double z_ui_prop_number(ZValue props,const char*key,double fallback){return z_ui_number(z_desktop_dict_get(props,key),fallback);}
static const char*z_ui_prop_string(ZValue props,const char*key,const char*fallback){return z_desktop_opt_string(props,key,fallback);}
static bool z_ui_prop_bool(ZValue props,const char*key,bool fallback){return z_desktop_opt_bool(props,key,fallback);}
static bool z_ui_scrollable_y(ZUINodeNative*n){if(n==NULL)return false;if(z_ui_prop_bool(n->props,"scrollY",false))return true;const char*overflow=z_ui_prop_string(n->props,"overflowY",z_ui_prop_string(n->props,"overflow",""));return strcasecmp(overflow,"scroll")==0||strcasecmp(overflow,"auto")==0;}
static bool z_ui_clips_children(ZUINodeNative*n){if(n==NULL)return false;if(z_ui_scrollable_y(n)||z_ui_prop_bool(n->props,"clipChildren",false))return true;const char*overflow=z_ui_prop_string(n->props,"overflowY",z_ui_prop_string(n->props,"overflow",""));return strcasecmp(overflow,"hidden")==0||strcasecmp(overflow,"clip")==0;}
static double z_ui_clamp(double value,double minimum,double maximum){if(value<minimum)return minimum;if(value>maximum)return maximum;return value;}
static bool z_ui_rect_intersection(ZUIRectNative a,ZUIRectNative b,ZUIRectNative*out){double x1=fmax(a.x,b.x),y1=fmax(a.y,b.y),x2=fmin(a.x+a.width,b.x+b.width),y2=fmin(a.y+a.height,b.y+b.height);if(x2<=x1||y2<=y1)return false;if(out)*out=(ZUIRectNative){x1,y1,x2-x1,y2-y1};return true;}
static bool z_ui_point_in_rect(double x,double y,ZUIRectNative b){return x>=b.x&&y>=b.y&&x<=b.x+b.width&&y<=b.y+b.height;}
static void z_ui_dict_set(ZValue dict,const char*key,ZValue value){z_set_index(dict,z_string(key),value);}
static ZValue z_ui_theme_values(bool dark){
    ZValue pairs[18]={
        z_pair(z_string("background"),z_string(dark?"#141822":"#f5f7fb")),
        z_pair(z_string("surface"),z_string(dark?"#202633":"#ffffff")),
        z_pair(z_string("surfaceAlt"),z_string(dark?"#2a3242":"#eef2f7")),
        z_pair(z_string("text"),z_string(dark?"#f2f5fa":"#172033")),
        z_pair(z_string("muted"),z_string(dark?"#a9b4c7":"#697386")),
        z_pair(z_string("primary"),z_string("#3867e8")),
        z_pair(z_string("primaryText"),z_string("#ffffff")),
        z_pair(z_string("border"),z_string(dark?"#465066":"#cfd6e2")),
        z_pair(z_string("danger"),z_string("#c73737")),
        z_pair(z_string("focus"),z_string("#6e95ff")),
        z_pair(z_string("radius"),z_int(6)),
        z_pair(z_string("fontFamily"),z_string("sans")),
        z_pair(z_string("fontPath"),z_string("")),
        z_pair(z_string("fontWeight"),z_string("normal")),
        z_pair(z_string("fontStyle"),z_string("normal")),
        z_pair(z_string("fontSize"),z_int(14)),
        z_pair(z_string("lineHeight"),z_float(1.25)),
        z_pair(z_string("spacing"),z_int(8))
    };
    ZValue result=z_dict_from(pairs,18);
    z_ui_dict_set(result,"controlHeight",z_int(36));
    return result;
}
static ZUIThemeNative*z_ui_theme_new(ZValue source){
    ZUIThemeNative*t=(ZUIThemeNative*)z_alloc(sizeof(*t));memset(t,0,sizeof(*t));
    if(source.tag==ZV_STRING){t->name=z_strdup(source.as.s);t->values=z_ui_theme_values(strcasecmp(source.as.s,"dark")==0);}
    else if(source.tag==ZV_DICT){t->name=z_strdup("custom");t->values=z_ui_theme_values(false);for(size_t i=0;i<source.as.dict->len;i++)z_set_index(t->values,source.as.dict->keys[i],source.as.dict->values[i]);}
    else { z_fatal("uiTheme expects string or dictionary"); }
    return t;
}
static ZUINodeNative*z_ui_node_new(const char*kind,ZValue props,ZValue children){
    if(props.tag!=ZV_DICT) { z_fatal("UI node props must be dictionary"); }
    if(children.tag!=ZV_ARRAY) { z_fatal("UI node children must be array"); }
    ZValue nested=z_desktop_dict_get(props,"options");if(nested.tag==ZV_DICT){for(size_t i=0;i<nested.as.dict->len;i++){ZValue key=nested.as.dict->keys[i];if(key.tag==ZV_STRING&&z_desktop_dict_get(props,key.as.s).tag==ZV_NULL)z_set_index(props,key,nested.as.dict->values[i]);}}
    ZUINodeNative*n=(ZUINodeNative*)z_alloc(sizeof(*n));memset(n,0,sizeof(*n));n->kind=z_strdup(kind);n->props=props;n->children=children;n->visible=z_ui_prop_bool(props,"visible",true);n->enabled=!z_ui_prop_bool(props,"disabled",false);
    const char*id=z_ui_prop_string(props,"id","");char generated[64];if(*id=='\0'){snprintf(generated,sizeof(generated),"ui-%" PRId64,z_ui_next_id++);id=generated;}n->id=z_strdup(id);
    for(size_t i=0;i<children.as.array->len;i++){ZUINodeNative*c=z_ui_expect_node(children.as.array->items[i]);c->parent=n;}return n;
}
static void z_ui_attach_tree(ZUIContextNative*ctx,ZUINodeNative*n){if(n==NULL)return;n->context=ctx;for(size_t i=0;i<n->children.as.array->len;i++)z_ui_attach_tree(ctx,z_ui_expect_node(n->children.as.array->items[i]));}
static double z_ui_theme_number(ZUIThemeNative*t,const char*key,double fallback){return t?z_ui_prop_number(t->values,key,fallback):fallback;}
static const char*z_ui_theme_string(ZUIThemeNative*t,const char*key,const char*fallback){return t?z_ui_prop_string(t->values,key,fallback):fallback;}
static size_t z_ui_array_length(ZValue value){return value.tag==ZV_ARRAY?value.as.array->len:0;}
typedef struct {double left,top,right,bottom;} ZUIEdgesNative;
static ZUIEdgesNative z_ui_edges(ZValue props,const char*key,double fallback){
    ZUIEdgesNative e={fallback,fallback,fallback,fallback};ZValue all=z_desktop_dict_get(props,key);if(all.tag!=ZV_NULL){double v=z_ui_number(all,fallback);e=(ZUIEdgesNative){v,v,v,v};}
    char name[96];snprintf(name,sizeof(name),"%sLeft",key);e.left=z_ui_prop_number(props,name,e.left);snprintf(name,sizeof(name),"%sTop",key);e.top=z_ui_prop_number(props,name,e.top);snprintf(name,sizeof(name),"%sRight",key);e.right=z_ui_prop_number(props,name,e.right);snprintf(name,sizeof(name),"%sBottom",key);e.bottom=z_ui_prop_number(props,name,e.bottom);return e;
}
static const char*z_ui_props_text(ZValue props){
    const char*keys[]={"text","label","value","placeholder","title",NULL};
    for(size_t i=0;keys[i]!=NULL;i++){
        ZValue value=z_desktop_dict_get(props,keys[i]);
        if(value.tag==ZV_STRING&&value.as.s!=NULL&&*value.as.s!='\0')return value.as.s;
    }
    return "";
}
static double z_ui_natural_height(ZUINodeNative*n,ZUIThemeNative*t){
    double control=z_ui_theme_number(t,"controlHeight",36);
    double font=z_ui_prop_number(n->props,"fontSize",z_ui_theme_number(t,"fontSize",14));
    double text_width=0,text_height=0;
    z_ui_measure_text_native(z_ui_props_text(n->props),n->props,t,z_ui_prop_number(n->props,"wrapWidth",0),&text_width,&text_height);
    (void)text_width;
    if(strcmp(n->kind,"text")==0)return fmax(font+8,text_height+8);
    if(strcmp(n->kind,"textarea")==0){double rows=z_ui_prop_number(n->props,"rows",4),line_height=z_ui_prop_number(n->props,"lineHeight",z_ui_theme_number(t,"lineHeight",1.25));return fmax(rows*font*line_height+16,control);}
    if(strcmp(n->kind,"table")==0){size_t rows=z_ui_array_length(z_desktop_dict_get(n->props,"rows"));if(rows==0)rows=1;return(double)(rows+1)*control;}
    if(strcmp(n->kind,"list")==0||strcmp(n->kind,"tree")==0){size_t items=z_ui_array_length(z_desktop_dict_get(n->props,"items"));if(items==0)items=1;return(double)items*control;}
    if(strcmp(n->kind,"tabs")==0)return control;
    if(strcmp(n->kind,"menu")==0){const char*placement=z_ui_prop_string(n->props,"placement","left");if(strcmp(placement,"top")==0||strcmp(placement,"bottom")==0){double size=z_ui_prop_number(n->props,"expandedSize",control);if(z_ui_prop_bool(n->props,"collapsed",false))size=z_ui_prop_number(n->props,"collapsedSize",fmin(size,48));return size;}return z_ui_prop_number(n->props,"height",control);}
    if(strcmp(n->kind,"modal")==0)return z_ui_prop_number(n->props,"contentHeight",320);
    if(strcmp(n->kind,"tooltip")==0)return fmax(font+16,text_height+12);
    if(strcmp(n->kind,"progress")==0)return 20;
    if(strcmp(n->kind,"image")==0||strcmp(n->kind,"canvas")==0)return z_ui_prop_number(n->props,"height",160);
    if(strcmp(n->kind,"spacer")==0)return z_ui_prop_number(n->props,"size",8);
    return z_ui_prop_number(n->props,"height",control);
}
static double z_ui_natural_width(ZUINodeNative*n,ZUIThemeNative*t){
    double text_width=0,text_height=0;
    z_ui_measure_text_native(z_ui_props_text(n->props),n->props,t,0,&text_width,&text_height);
    (void)text_height;
    double width=text_width+24;if(width<48)width=48;
    if(strcmp(n->kind,"input")==0||strcmp(n->kind,"textarea")==0||strcmp(n->kind,"select")==0)if(width<180)width=180;
    if(strcmp(n->kind,"checkbox")==0||strcmp(n->kind,"radio")==0)width+=20;
    if(strcmp(n->kind,"image")==0||strcmp(n->kind,"canvas")==0)width=z_ui_prop_number(n->props,"width",240);
    if(strcmp(n->kind,"menu")==0){const char*placement=z_ui_prop_string(n->props,"placement","left");if(strcmp(placement,"left")==0||strcmp(placement,"right")==0){width=z_ui_prop_number(n->props,"expandedSize",z_ui_prop_number(n->props,"width",240));if(z_ui_prop_bool(n->props,"collapsed",false))width=z_ui_prop_number(n->props,"collapsedSize",fmin(width,56));}else width=z_ui_prop_number(n->props,"width",width);}
    if(strcmp(n->kind,"spacer")==0)width=z_ui_prop_number(n->props,"size",8);
    return width;
}
static void z_ui_layout(ZUINodeNative*n,ZUIRectNative available,ZUIThemeNative*t){
    if(n==NULL)return;
    if(!n->visible){n->bounds=(ZUIRectNative){0,0,0,0};n->content_bounds=(ZUIRectNative){0,0,0,0};n->content_height=0;n->scroll_offset_y=0;return;}
    ZUIEdgesNative margin=z_ui_edges(n->props,"margin",0);
    double x=available.x+margin.left,y=available.y+margin.top,width=available.width-margin.left-margin.right,height=available.height-margin.top-margin.bottom;
    ZValue wv=z_desktop_dict_get(n->props,"width"),hv=z_desktop_dict_get(n->props,"height");
    if(strcmp(n->kind,"modal")!=0){if(wv.tag!=ZV_NULL)width=z_ui_number(wv,width);if(hv.tag!=ZV_NULL)height=z_ui_number(hv,height);}
    double minw=z_ui_prop_number(n->props,"minWidth",width),maxw=z_ui_prop_number(n->props,"maxWidth",width),minh=z_ui_prop_number(n->props,"minHeight",height),maxh=z_ui_prop_number(n->props,"maxHeight",height);
    if(width<minw)width=minw;if(width>maxw)width=maxw;if(height<minh)height=minh;if(height>maxh)height=maxh;if(width<0)width=0;if(height<0)height=0;
    if(hv.tag==ZV_NULL&&strcmp(n->kind,"row")!=0&&strcmp(n->kind,"column")!=0&&strcmp(n->kind,"container")!=0&&strcmp(n->kind,"menu")!=0&&strcmp(n->kind,"modal")!=0){double natural=z_ui_natural_height(n,t);if(height>natural)height=natural;}
    if(strcmp(n->kind,"menu")==0){const char*placement=z_ui_prop_string(n->props,"placement","left");bool collapsed=z_ui_prop_bool(n->props,"collapsed",false);if(strcmp(placement,"top")==0||strcmp(placement,"bottom")==0){double size=z_ui_prop_number(n->props,"expandedSize",height);if(collapsed)size=z_ui_prop_number(n->props,"collapsedSize",fmin(size,48));height=size;}else{double size=z_ui_prop_number(n->props,"expandedSize",width);if(collapsed)size=z_ui_prop_number(n->props,"collapsedSize",fmin(size,56));width=size;}}
    if(strcmp(n->kind,"modal")==0){double margin=fmax(0,z_ui_prop_number(n->props,"viewportMargin",24));double mw=z_ui_prop_number(n->props,"contentWidth",z_ui_prop_number(n->props,"width",480)),mh=z_ui_prop_number(n->props,"contentHeight",z_ui_prop_number(n->props,"height",320));double maxw=fmax(0,width-margin*2),maxh=fmax(0,height-margin*2);if(mw>maxw)mw=maxw;if(mh>maxh)mh=maxh;x+=(width-mw)/2;y+=(height-mh)/2;width=mw;height=mh;}
    n->bounds=(ZUIRectNative){x,y,width,height};
    ZUIEdgesNative padding=z_ui_edges(n->props,"padding",0);
    ZUIRectNative inner={x+padding.left,y+padding.top,width-padding.left-padding.right,height-padding.top-padding.bottom};if(inner.width<0)inner.width=0;if(inner.height<0)inner.height=0;
    n->content_bounds=inner;n->content_height=inner.height;

    size_t child_count=n->children.as.array->len;
    ZUINodeNative**flow=child_count==0?NULL:(ZUINodeNative**)malloc(sizeof(*flow)*child_count);
    ZUINodeNative**overlays=child_count==0?NULL:(ZUINodeNative**)malloc(sizeof(*overlays)*child_count);
    if(child_count!=0&&(flow==NULL||overlays==NULL)){free(flow);free(overlays);z_fatal("out of memory laying out UI");}
    size_t flow_count=0,overlay_count=0;
    for(size_t i=0;i<child_count;i++){ZUINodeNative*c=z_ui_expect_node(n->children.as.array->items[i]);if(!c->visible){c->bounds=(ZUIRectNative){0,0,0,0};c->content_bounds=(ZUIRectNative){0,0,0,0};c->content_height=0;c->scroll_offset_y=0;continue;}if(strcmp(c->kind,"modal")==0)overlays[overlay_count++]=c;else flow[flow_count++]=c;}
    if(flow_count==0){n->scroll_offset_y=0;for(size_t i=0;i<overlay_count;i++)z_ui_layout(overlays[i],n->bounds,t);free(flow);free(overlays);return;}

    const char*direction=n->kind;if(strcmp(direction,"row")!=0&&strcmp(direction,"column")!=0)direction=z_ui_prop_string(n->props,"direction","column");if(strcmp(n->kind,"menu")==0){const char*placement=z_ui_prop_string(n->props,"placement","left");direction=(strcmp(placement,"top")==0||strcmp(placement,"bottom")==0)?"row":"column";}if(strcmp(n->kind,"modal")==0)direction="column";bool row=strcmp(direction,"row")==0;
    double gap=z_ui_prop_number(n->props,"gap",z_ui_theme_number(t,"spacing",8));if(gap<0)gap=0;
    double main=(row?inner.width:inner.height)-gap*(double)(flow_count>0?flow_count-1:0);if(main<0)main=0;double fixed=0,grow_total=0;
    for(size_t i=0;i<flow_count;i++){ZUINodeNative*c=flow[i];double grow=z_ui_prop_number(c->props,"grow",0);if(grow>0)grow_total+=grow;else fixed+=row?z_ui_prop_number(c->props,"width",z_ui_natural_width(c,t)):z_ui_prop_number(c->props,"height",z_ui_natural_height(c,t));}
    double remaining=main-fixed;if(remaining<0)remaining=0;
    double content_height=inner.height;if(!row){double used=fixed+gap*(double)(flow_count>0?flow_count-1:0);if(grow_total>0)used+=remaining;if(used>content_height)content_height=used;}
    bool scrollable=z_ui_scrollable_y(n)&&!row;
    if(!scrollable)n->scroll_offset_y=0;else{double max_offset=fmax(0,content_height-inner.height);if(max_offset<=0.5)max_offset=0;n->scroll_offset_y=z_ui_clamp(n->scroll_offset_y,0,max_offset);if(max_offset>0&&(!z_ui_prop_bool(n->props,"scrollbarOverlay",false)||z_ui_prop_bool(n->props,"scrollbarAvoidContent",false))){double bar=fmax(4,z_ui_prop_number(n->props,"scrollbarWidth",8)),gutter=fmax(2,z_ui_prop_number(n->props,"scrollbarGutter",4));inner.width=fmax(0,inner.width-bar-gutter);}}
    n->content_bounds=inner;n->content_height=content_height;
    double cursor_x=inner.x,cursor_y=inner.y-(scrollable?n->scroll_offset_y:0);const char*justify=z_ui_prop_string(n->props,"justify","start");
    if(grow_total==0&&remaining>0){if(strcmp(justify,"center")==0){if(row)cursor_x+=remaining/2;else cursor_y+=remaining/2;}else if(strcmp(justify,"end")==0){if(row)cursor_x+=remaining;else cursor_y+=remaining;}else if(strcmp(justify,"space-between")==0&&flow_count>1)gap+=remaining/(double)(flow_count-1);}
    for(size_t i=0;i<flow_count;i++){ZUINodeNative*c=flow[i];double grow=z_ui_prop_number(c->props,"grow",0),cw,ch;if(row){cw=grow>0&&grow_total>0?remaining*grow/grow_total:z_ui_prop_number(c->props,"width",z_ui_natural_width(c,t));ch=inner.height;const char*align=z_ui_prop_string(n->props,"align","stretch");double natural=z_ui_natural_height(c,t);if(strcmp(align,"stretch")!=0){ch=z_ui_prop_number(c->props,"height",natural);if(ch>inner.height)ch=inner.height;}double cy=cursor_y;if(strcmp(align,"center")==0)cy=inner.y+(inner.height-ch)/2;else if(strcmp(align,"end")==0)cy=inner.y+inner.height-ch;z_ui_layout(c,(ZUIRectNative){cursor_x,cy,cw,ch},t);cursor_x+=cw+gap;}else{ch=grow>0&&grow_total>0?remaining*grow/grow_total:z_ui_prop_number(c->props,"height",z_ui_natural_height(c,t));cw=inner.width;const char*align=z_ui_prop_string(n->props,"align","stretch");double natural=z_ui_natural_width(c,t);if(strcmp(align,"stretch")!=0){cw=z_ui_prop_number(c->props,"width",natural);if(cw>inner.width)cw=inner.width;}double cx=cursor_x;if(strcmp(align,"center")==0)cx=inner.x+(inner.width-cw)/2;else if(strcmp(align,"end")==0)cx=inner.x+inner.width-cw;z_ui_layout(c,(ZUIRectNative){cx,cursor_y,cw,ch},t);cursor_y+=ch+gap;}}
    for(size_t i=0;i<overlay_count;i++)z_ui_layout(overlays[i],n->bounds,t);
    free(flow);free(overlays);
}

static void z_ui_color_parse(const char*s,uint8_t*r,uint8_t*g,uint8_t*b,uint8_t*a){*r=*g=*b=0;*a=255;if(s==NULL||strcmp(s,"transparent")==0){*a=0;return;}if(*s=='#')s++;size_t n=strlen(s);unsigned long v=strtoul(s,NULL,16);if(n==6){*r=(v>>16)&255;*g=(v>>8)&255;*b=v&255;}else if(n==8){*r=(v>>24)&255;*g=(v>>16)&255;*b=(v>>8)&255;*a=v&255;}}
#if defined(ZUMBRA_DESKTOP_SDL3)
static void z_ui_draw_color(SDL_Renderer*r,const char*color){uint8_t red,green,blue,alpha;z_ui_color_parse(color,&red,&green,&blue,&alpha);z_desktop_sdl.SetRenderDrawColor(r,red,green,blue,alpha);}
static void z_ui_fill(SDL_Renderer*r,ZUIRectNative q,const char*color){uint8_t a,b,c,d;z_ui_color_parse(color,&a,&b,&c,&d);if(d==0)return;z_desktop_sdl.SetRenderDrawColor(r,a,b,c,d);ZDesktopSDLFRect f={(float)q.x,(float)q.y,(float)q.width,(float)q.height};z_desktop_sdl.RenderFillRect(r,&f);}
static void z_ui_stroke(SDL_Renderer*r,ZUIRectNative q,const char*color){z_ui_draw_color(r,color);ZDesktopSDLFRect f={(float)q.x,(float)q.y,(float)q.width,(float)q.height};z_desktop_sdl.RenderRect(r,&f);}
static void z_ui_text(SDL_Renderer*r,double x,double y,const char*text,const char*color,ZValue props,ZUIThemeNative*theme){
    if(text==NULL||*text=='\0')return;
    double size=z_ui_prop_number(props,"fontSize",z_ui_theme_number(theme,"fontSize",14));
    const char*family=z_ui_prop_string(props,"fontFamily",z_ui_theme_string(theme,"fontFamily","sans"));
    const char*path=z_ui_prop_string(props,"fontPath",z_ui_theme_string(theme,"fontPath",""));
    const char*weight=z_ui_prop_string(props,"fontWeight",z_ui_theme_string(theme,"fontWeight","normal"));
    const char*style=z_ui_prop_string(props,"fontStyle",z_ui_theme_string(theme,"fontStyle","normal"));
    double wrap_value=z_ui_prop_number(props,"wrapWidth",0);
    int wrap=wrap_value>0?(int)ceil(wrap_value):0;
    uint8_t red,green,blue,alpha;z_ui_color_parse(color,&red,&green,&blue,&alpha);
    TTF_Font*font=z_ui_font(family,path,size,weight,style);
    if(font&&z_desktop_sdl.CreateTextureFromSurface&&z_desktop_sdl.RenderTexture&&z_desktop_sdl.DestroyTexture&&z_desktop_sdl.DestroySurface){
        uint32_t rgba=z_ui_rgba_key(red,green,blue,alpha);
        ZUITextTextureCacheNative*cached=z_ui_text_cache_find(r,font,text,rgba,wrap);
        if(cached&&cached->texture){ZDesktopSDLFRect dst={(float)x,(float)y,cached->width,cached->height};z_desktop_sdl.RenderTexture(r,cached->texture,NULL,&dst);return;}
        ZUITTFColor c={red,green,blue,alpha};
        SDL_Surface*surface=(wrap>0&&z_ui_ttf.RenderTextBlendedWrapped)?z_ui_ttf.RenderTextBlendedWrapped(font,text,0,c,wrap):z_ui_ttf.RenderTextBlended(font,text,0,c);
        if(surface){
            SDL_Texture*texture=z_desktop_sdl.CreateTextureFromSurface(r,surface);
            z_desktop_sdl.DestroySurface(surface);
            if(texture){
                float width=0,height=0;
                if(z_desktop_sdl.GetTextureSize)z_desktop_sdl.GetTextureSize(texture,&width,&height);
                if(width<=0||height<=0){double mw=0,mh=0;z_ui_measure_text_native(text,props,theme,wrap_value,&mw,&mh);width=(float)mw;height=(float)mh;}
                ZUITextTextureCacheNative*stored=z_ui_text_cache_store(r,font,text,rgba,wrap,texture,width,height);
                ZDesktopSDLFRect dst={(float)x,(float)y,width,height};
                z_desktop_sdl.RenderTexture(r,texture,NULL,&dst);
                if(stored==NULL)z_ui_destroy_texture(texture);
                return;
            }
        }
    }
    if(z_desktop_sdl.RenderDebugText){z_desktop_sdl.SetRenderDrawColor(r,red,green,blue,alpha);z_desktop_sdl.RenderDebugText(r,(float)x,(float)y,text);}
}
static size_t z_ui_utf8_byte_offset(const char*text,size_t index);
/* Temporary fitted strings are released immediately after painting. They must
 * use the system allocator rather than z_alloc(), whose allocations are owned
 * by the runtime and released together during z_runtime_shutdown(). */
static char*z_ui_temporary_string(const char*text){
    if(text==NULL)text="";
    size_t length=strlen(text);
    char*copy=(char*)malloc(length+1);
    if(copy==NULL)return NULL;
    memcpy(copy,text,length+1);
    return copy;
}
static char*z_ui_fit_text_native(const char*text,double max_width,ZValue props,ZUIThemeNative*theme){
    if(!text)text="";
    double width=0,height=0;z_ui_measure_text_native(text,props,theme,z_ui_prop_number(props,"wrapWidth",0),&width,&height);
    if(max_width<=0)return z_ui_temporary_string("");
    if(width<=max_width||strcasecmp(z_ui_prop_string(props,"textOverflow","ellipsis"),"visible")==0)return z_ui_temporary_string(text);
    const char*ellipsis="…";double ellipsis_width=0;z_ui_measure_text_native(ellipsis,props,theme,0,&ellipsis_width,&height);
    if(ellipsis_width>max_width)return z_ui_temporary_string("");
    size_t count=z_ui_utf8_count(text),ellipsis_bytes=strlen(ellipsis);
    while(count>0){
        size_t bytes=z_ui_utf8_byte_offset(text,count);char*candidate=(char*)malloc(bytes+ellipsis_bytes+1);
        if(!candidate)return z_ui_temporary_string(ellipsis);
        memcpy(candidate,text,bytes);memcpy(candidate+bytes,ellipsis,ellipsis_bytes+1);
        z_ui_measure_text_native(candidate,props,theme,0,&width,&height);
        if(width<=max_width)return candidate;
        free(candidate);count--;
    }
    return z_ui_temporary_string(ellipsis);
}
static void z_ui_text_aligned(SDL_Renderer*r,ZUIRectNative bounds,double padding,const char*text,const char*color,ZValue props,ZUIThemeNative*theme,const char*default_align){
    double available=fmax(0,bounds.width-padding*2);char*fitted=z_ui_fit_text_native(text,available,props,theme);if(fitted==NULL)return;double width=0,height=0;
    z_ui_measure_text_native(fitted,props,theme,z_ui_prop_number(props,"wrapWidth",0),&width,&height);
    double y=bounds.y+fmax(0,(bounds.height-height)/2),x=bounds.x+padding;const char*align=z_ui_prop_string(props,"textAlign",default_align);
    if(strcmp(align,"center")==0)x=bounds.x+fmax(padding,(bounds.width-width)/2);else if(strcmp(align,"right")==0)x=bounds.x+fmax(padding,bounds.width-padding-width);
    z_ui_text(r,x,y,fitted,color,props,theme);free(fitted);
}
static void z_ui_text_centered(SDL_Renderer*r,ZUIRectNative bounds,double left_padding,const char*text,const char*color,ZValue props,ZUIThemeNative*theme){z_ui_text_aligned(r,bounds,left_padding,text,color,props,theme,"left");}
static bool z_ui_render_control_icon(SDL_Renderer*r,ZUIRectNative b,const char*icon,const char*color){
    if(icon==NULL||*icon=='\0')return false;double cx=b.x+b.width/2,cy=b.y+b.height/2,size=fmax(4,fmin(b.width,b.height)*.22);z_ui_draw_color(r,color);
    if(strcasecmp(icon,"close")==0||strcasecmp(icon,"x")==0){z_desktop_sdl.RenderLine(r,(float)(cx-size),(float)(cy-size),(float)(cx+size),(float)(cy+size));z_desktop_sdl.RenderLine(r,(float)(cx+size),(float)(cy-size),(float)(cx-size),(float)(cy+size));return true;}
    if(strcasecmp(icon,"menu")==0||strcasecmp(icon,"hamburger")==0){for(int i=-1;i<=1;i++)z_desktop_sdl.RenderLine(r,(float)(cx-size),(float)(cy+(double)i*size),(float)(cx+size),(float)(cy+(double)i*size));return true;}
    if(strcasecmp(icon,"chevron-left")==0||strcasecmp(icon,"collapse-left")==0){z_desktop_sdl.RenderLine(r,(float)(cx+size/2),(float)(cy-size),(float)(cx-size/2),(float)cy);z_desktop_sdl.RenderLine(r,(float)(cx-size/2),(float)cy,(float)(cx+size/2),(float)(cy+size));return true;}
    if(strcasecmp(icon,"chevron-right")==0||strcasecmp(icon,"expand-right")==0){z_desktop_sdl.RenderLine(r,(float)(cx-size/2),(float)(cy-size),(float)(cx+size/2),(float)cy);z_desktop_sdl.RenderLine(r,(float)(cx+size/2),(float)cy,(float)(cx-size/2),(float)(cy+size));return true;}
    return false;
}
static const char*z_ui_node_text(ZUINodeNative*n){return z_ui_props_text(n->props);}
static const char*z_ui_value_text(ZValue value){if(value.tag==ZV_STRING)return value.as.s;ZValue converted=z_to_string_value(value);return converted.tag==ZV_STRING?converted.as.s:"";}
static void z_ui_set_clip(SDL_Renderer*r,bool enabled,ZUIRectNative clip){if(z_desktop_sdl.SetRenderClipRect==NULL)return;if(!enabled){z_desktop_sdl.SetRenderClipRect(r,NULL);return;}ZDesktopSDLRect q={(int32_t)floor(clip.x),(int32_t)floor(clip.y),(int32_t)ceil(clip.width),(int32_t)ceil(clip.height)};z_desktop_sdl.SetRenderClipRect(r,&q);}
static bool z_ui_set_local_clip(SDL_Renderer*r,bool has_parent,ZUIRectNative parent,ZUIRectNative local){
    ZUIRectNative effective=local;
    if(has_parent&&!z_ui_rect_intersection(parent,local,&effective))return false;
    z_ui_set_clip(r,true,effective);
    return true;
}

static const char*z_ui_chart_color(ZValue colors,size_t index){static const char*palette[]={"#3867e8","#e89b38","#42a56b","#9b59b6","#e05260","#2f9db0","#8a6d3b","#6c7a89"};if(colors.tag==ZV_ARRAY&&colors.as.array->len>0)return z_ui_value_text(colors.as.array->items[index%colors.as.array->len]);return palette[index%(sizeof(palette)/sizeof(palette[0]))];}
static void z_ui_render_pie_chart(SDL_Renderer*r,ZUIRectNative origin,ZValue values,ZUIThemeNative*theme){ZValue numbers=z_desktop_dict_get(values,"values"),colors=z_desktop_dict_get(values,"colors"),labels=z_desktop_dict_get(values,"labels");if(numbers.tag!=ZV_ARRAY||numbers.as.array->len==0)return;double x=origin.x+z_ui_prop_number(values,"x",0),y=origin.y+z_ui_prop_number(values,"y",0),width=z_ui_prop_number(values,"width",origin.width),height=z_ui_prop_number(values,"height",origin.height);bool legend=z_ui_prop_bool(values,"legend",true);double legend_width=legend?fmin(170,width*.42):0,plot_width=fmax(0,width-legend_width),radius=fmax(0,fmin(plot_width,height)/2-8),cx=x+plot_width/2,cy=y+height/2,total=0;for(size_t i=0;i<numbers.as.array->len;i++){double n=z_ui_number(numbers.as.array->items[i],0);if(n>0)total+=n;}if(total<=0||radius<=0)return;double start=-3.14159265358979323846/2;for(size_t i=0;i<numbers.as.array->len;i++){double n=z_ui_number(numbers.as.array->items[i],0);if(n<=0)continue;double end=start+n/total*2*3.14159265358979323846;z_ui_draw_color(r,z_ui_chart_color(colors,i));int steps=(int)fmax(2,ceil((end-start)*radius*1.6));for(int step=0;step<=steps;step++){double angle=start+(end-start)*(double)step/(double)steps,ex=cx+cos(angle)*radius,ey=cy+sin(angle)*radius;z_desktop_sdl.RenderLine(r,(float)cx,(float)cy,(float)ex,(float)ey);}start=end;}if(legend){double legend_x=x+plot_width+8;for(size_t i=0;i<numbers.as.array->len;i++){double row_y=y+8+(double)i*24;if(row_y+18>y+height)break;const char*color=z_ui_chart_color(colors,i);z_ui_fill(r,(ZUIRectNative){legend_x,row_y+3,12,12},color);const char*label=(labels.tag==ZV_ARRAY&&i<labels.as.array->len)?z_ui_value_text(labels.as.array->items[i]):"";char buffer[256];double n=z_ui_number(numbers.as.array->items[i],0);if(*label)snprintf(buffer,sizeof(buffer),"%s: %.0f",label,n);else snprintf(buffer,sizeof(buffer),"%.0f",n);z_ui_text(r,legend_x+18,row_y,buffer,z_ui_prop_string(values,"textColor",z_ui_theme_string(theme,"text","#172033")),values,theme);}}}
static void z_ui_render_bar_chart(SDL_Renderer*r,ZUIRectNative origin,ZValue values,ZUIThemeNative*theme){ZValue numbers=z_desktop_dict_get(values,"values"),colors=z_desktop_dict_get(values,"colors"),labels=z_desktop_dict_get(values,"labels");if(numbers.tag!=ZV_ARRAY||numbers.as.array->len==0)return;double x=origin.x+z_ui_prop_number(values,"x",0),y=origin.y+z_ui_prop_number(values,"y",0),width=z_ui_prop_number(values,"width",origin.width),height=z_ui_prop_number(values,"height",origin.height),padding=fmax(8,z_ui_prop_number(values,"padding",18)),label_h=28;ZUIRectNative plot={x+padding,y+padding,fmax(0,width-padding*2),fmax(0,height-padding*2-label_h)};double max_value=0;for(size_t i=0;i<numbers.as.array->len;i++){double n=z_ui_number(numbers.as.array->items[i],0);if(n>max_value)max_value=n;}if(max_value<=0||plot.width<=0||plot.height<=0)return;double gap=fmax(4,z_ui_prop_number(values,"gap",8)),bar_width=fmax(2,(plot.width-gap*(numbers.as.array->len+1))/(double)numbers.as.array->len);for(size_t i=0;i<numbers.as.array->len;i++){double n=fmax(0,z_ui_number(numbers.as.array->items[i],0)),bar_h=plot.height*n/max_value,bar_x=plot.x+gap+(double)i*(bar_width+gap);ZUIRectNative bar={bar_x,plot.y+plot.height-bar_h,bar_width,bar_h};z_ui_fill(r,bar,z_ui_chart_color(colors,i));char value_text[64];snprintf(value_text,sizeof(value_text),"%.0f",n);z_ui_text_aligned(r,(ZUIRectNative){bar_x,bar.y-22,bar_width,20},2,value_text,z_ui_prop_string(values,"textColor",z_ui_theme_string(theme,"text","#172033")),values,theme,"center");if(labels.tag==ZV_ARRAY&&i<labels.as.array->len){z_ui_text_aligned(r,(ZUIRectNative){bar_x,plot.y+plot.height+4,bar_width,label_h},2,z_ui_value_text(labels.as.array->items[i]),z_ui_prop_string(values,"textColor",z_ui_theme_string(theme,"text","#172033")),values,theme,"center");}}}
static void z_ui_render_line_chart(SDL_Renderer*r,ZUIRectNative origin,ZValue values,ZUIThemeNative*theme){
    ZValue numbers=z_desktop_dict_get(values,"values"),labels=z_desktop_dict_get(values,"labels");
    if(numbers.tag!=ZV_ARRAY||numbers.as.array->len==0)return;
    double x=origin.x+z_ui_prop_number(values,"x",0),y=origin.y+z_ui_prop_number(values,"y",0),width=z_ui_prop_number(values,"width",origin.width),height=z_ui_prop_number(values,"height",origin.height),padding=fmax(8,z_ui_prop_number(values,"padding",18));
    bool has_labels=labels.tag==ZV_ARRAY&&labels.as.array->len>0;
    double label_h=has_labels?30:0,value_h=z_ui_prop_bool(values,"showValues",true)?24:0;
    ZUIRectNative plot={x+padding,y+padding+value_h,fmax(0,width-padding*2),fmax(0,height-padding*2-label_h-value_h)};
    if(plot.width<=0||plot.height<=0)return;
    double min_value=z_ui_number(numbers.as.array->items[0],0),max_value=min_value;
    for(size_t i=1;i<numbers.as.array->len;i++){double n=z_ui_number(numbers.as.array->items[i],0);if(n<min_value)min_value=n;if(n>max_value)max_value=n;}
    if(max_value==min_value){min_value-=.5;max_value+=.5;}
    const char*color=z_ui_prop_string(values,"color",z_ui_chart_color(z_desktop_dict_get(values,"colors"),0));
    const char*text_color=z_ui_prop_string(values,"textColor",z_ui_theme_string(theme,"text","#172033"));
    double slot=plot.width/(double)numbers.as.array->len;
    double previous_x=0,previous_y=0;
    for(size_t i=0;i<numbers.as.array->len;i++){
        double n=z_ui_number(numbers.as.array->items[i],0),px=plot.x+((double)i+.5)*slot,py=plot.y+plot.height-(n-min_value)/(max_value-min_value)*plot.height;
        if(i>0){z_ui_draw_color(r,color);z_desktop_sdl.RenderLine(r,(float)previous_x,(float)previous_y,(float)px,(float)py);}
        z_ui_fill(r,(ZUIRectNative){px-3,py-3,6,6},color);
        if(value_h>0){char value_text[64];snprintf(value_text,sizeof(value_text),"%.0f",n);double cell_w=fmax(48,slot);z_ui_text_aligned(r,(ZUIRectNative){px-cell_w/2,py-24,cell_w,20},2,value_text,text_color,values,theme,"center");}
        if(has_labels&&i<labels.as.array->len){double cell_w=fmax(48,slot);z_ui_text_aligned(r,(ZUIRectNative){px-cell_w/2,plot.y+plot.height+5,cell_w,label_h-4},2,z_ui_value_text(labels.as.array->items[i]),text_color,values,theme,"center");}
        previous_x=px;previous_y=py;
    }
}

static void z_ui_call_prop(ZUINodeNative*n,const char*name,ZValue event);
static void z_ui_update_bound(ZUINodeNative*n,const char*property,ZValue value);
static int64_t z_ui_select_current_index(ZUINodeNative*n,ZValue options){
    const char*value=z_ui_prop_string(n->props,"value","");
    if(options.tag==ZV_ARRAY){for(size_t i=0;i<options.as.array->len;i++)if(strcmp(z_ui_value_text(options.as.array->items[i]),value)==0)return(int64_t)i;}
    int64_t index=(int64_t)z_ui_prop_number(n->props,"selectedIndex",0);
    if(options.tag!=ZV_ARRAY||options.as.array->len==0)return 0;
    if(index<0||index>=(int64_t)options.as.array->len)return 0;
    return index;
}
static ZUIRectNative z_ui_select_popup_geometry(ZUINodeNative*n,ZUIRectNative viewport,double*row_height,size_t*visible_rows){
    ZValue options=z_desktop_dict_get(n->props,"options");size_t count=options.tag==ZV_ARRAY?options.as.array->len:0;
    double row=fmax(28,z_ui_prop_number(n->props,"optionHeight",fmax(34,n->bounds.height)));
    size_t max_visible=(size_t)fmax(1,z_ui_prop_number(n->props,"maxVisibleOptions",8));size_t visible=count<max_visible?count:max_visible;
    double height=row*(double)visible,width=fmax(n->bounds.width,z_ui_prop_number(n->props,"dropdownWidth",n->bounds.width));
    double x=n->bounds.x;if(x+width>viewport.x+viewport.width)x=fmax(viewport.x,viewport.x+viewport.width-width);
    double y=n->bounds.y+n->bounds.height+2;if(y+height>viewport.y+viewport.height&&n->bounds.y-2-height>=viewport.y)y=n->bounds.y-2-height;
    if(y+height>viewport.y+viewport.height)height=fmax(row,viewport.y+viewport.height-y);
    if(row_height)*row_height=row;if(visible_rows)*visible_rows=visible;return(ZUIRectNative){x,y,width,height};
}
static ZUINodeNative*z_ui_topmost_open_select(ZUINodeNative*n){
    if(n==NULL||!n->visible)return NULL;for(size_t i=n->children.as.array->len;i>0;i--){ZUINodeNative*found=z_ui_topmost_open_select(z_ui_expect_node(n->children.as.array->items[i-1]));if(found)return found;}
    return strcmp(n->kind,"select")==0&&z_ui_prop_bool(n->props,"open",false)?n:NULL;
}
static bool z_ui_close_open_selects(ZUINodeNative*n,ZUINodeNative*except){
    if(n==NULL)return false;bool changed=false;if(n!=except&&strcmp(n->kind,"select")==0&&z_ui_prop_bool(n->props,"open",false)){z_ui_dict_set(n->props,"open",z_bool(false));changed=true;}
    for(size_t i=0;i<n->children.as.array->len;i++)if(z_ui_close_open_selects(z_ui_expect_node(n->children.as.array->items[i]),except))changed=true;return changed;
}
static void z_ui_select_set_value(ZUINodeNative*n,int64_t index,ZValue ev){
    ZValue options=z_desktop_dict_get(n->props,"options");if(options.tag!=ZV_ARRAY||options.as.array->len==0)return;if(index<0)index=0;if(index>=(int64_t)options.as.array->len)index=(int64_t)options.as.array->len-1;
    ZValue selected_index=z_int(index),selected_value=z_string(z_ui_value_text(options.as.array->items[index]));z_ui_dict_set(n->props,"selectedIndex",selected_index);z_ui_dict_set(n->props,"value",selected_value);z_ui_dict_set(n->props,"open",z_bool(false));z_ui_update_bound(n,"selectedIndex",selected_index);z_ui_update_bound(n,"value",selected_value);z_ui_call_prop(n,"onChange",ev);
}
static void z_ui_render_select_popup(SDL_Renderer*r,ZUINodeNative*n,ZUIContextNative*ctx,ZUIRectNative viewport){
    if(n==NULL||strcmp(n->kind,"select")!=0||!z_ui_prop_bool(n->props,"open",false))return;ZValue options=z_desktop_dict_get(n->props,"options");if(options.tag!=ZV_ARRAY||options.as.array->len==0)return;
    double row=0;size_t visible=0;ZUIRectNative popup=z_ui_select_popup_geometry(n,viewport,&row,&visible);int64_t offset=(int64_t)z_ui_prop_number(n->props,"popupOffset",0);int64_t max_offset=(int64_t)options.as.array->len-(int64_t)visible;if(max_offset<0)max_offset=0;if(offset<0)offset=0;if(offset>max_offset)offset=max_offset;
    z_ui_fill(r,(ZUIRectNative){popup.x+4,popup.y+5,popup.width,popup.height},z_ui_prop_string(n->props,"dropdownShadow","#00000030"));z_ui_fill(r,popup,z_ui_prop_string(n->props,"dropdownBackground",z_ui_prop_string(n->props,"background",z_ui_theme_string(ctx->theme,"surface","#ffffff"))));z_ui_stroke(r,popup,z_ui_prop_string(n->props,"dropdownBorderColor",z_ui_theme_string(ctx->theme,"border","#cfd6e2")));
    const char*selected=z_ui_prop_string(n->props,"value","");double content_width=popup.width-(options.as.array->len>visible?10:0);for(size_t row_index=0;row_index<visible&&(size_t)(offset+(int64_t)row_index)<options.as.array->len;row_index++){size_t index=(size_t)(offset+(int64_t)row_index);ZUIRectNative item={popup.x+1,popup.y+(double)row_index*row+1,content_width-2,row-2};const char*label=z_ui_value_text(options.as.array->items[index]);if(strcmp(label,selected)==0)z_ui_fill(r,item,z_ui_prop_string(n->props,"selectedOptionBackground",z_ui_theme_string(ctx->theme,"surfaceAlt","#eef2f7")));z_ui_text_centered(r,item,10,label,z_ui_prop_string(n->props,"textColor",z_ui_theme_string(ctx->theme,"text","#172033")),n->props,ctx->theme);if(row_index>0)z_ui_fill(r,(ZUIRectNative){popup.x+1,popup.y+(double)row_index*row,content_width-2,1},z_ui_prop_string(n->props,"optionSeparatorColor",z_ui_theme_string(ctx->theme,"border","#cfd6e2")));}
    if(options.as.array->len>visible){ZUIRectNative track={popup.x+popup.width-8,popup.y+3,5,popup.height-6};z_ui_fill(r,track,z_ui_prop_string(n->props,"scrollbarTrack","transparent"));double thumb_h=fmax(18,track.height*(double)visible/(double)options.as.array->len),thumb_y=track.y;if(max_offset>0)thumb_y+=(track.height-thumb_h)*(double)offset/(double)max_offset;z_ui_fill(r,(ZUIRectNative){track.x,thumb_y,track.width,thumb_h},z_ui_prop_string(n->props,"scrollbarThumb","#9aa7ba"));}
}
static void z_ui_render_open_selects(SDL_Renderer*r,ZUINodeNative*n,ZUIContextNative*ctx,ZUIRectNative viewport){if(n==NULL||!n->visible)return;for(size_t i=0;i<n->children.as.array->len;i++)z_ui_render_open_selects(r,z_ui_expect_node(n->children.as.array->items[i]),ctx,viewport);z_ui_render_select_popup(r,n,ctx,viewport);}


#define Z_UI_PIXELFORMAT_RGBA32 0x16762004u
typedef struct{uint32_t flags,format;int w,h,pitch;void*pixels;int refcount;void*reserved;}ZUISurfaceView;
typedef SDL_Surface*(*ZUI_IMG_Load)(const char*);
static void*z_ui_image_library=NULL;static ZUI_IMG_Load z_ui_img_load=NULL;
#if defined(__linux__)
typedef void ZUIGdkPixbuf;typedef void ZUIGError;
typedef ZUIGdkPixbuf*(*ZUI_GDK_NEW)(const char*,ZUIGError**);typedef int(*ZUI_GDK_INT)(const ZUIGdkPixbuf*);typedef unsigned char*(*ZUI_GDK_PIXELS)(const ZUIGdkPixbuf*);typedef void(*ZUI_GOBJECT_UNREF)(void*);
static void*z_ui_pixbuf_library=NULL,*z_ui_gobject_library=NULL;static ZUI_GDK_NEW z_ui_gdk_new=NULL;static ZUI_GDK_INT z_ui_gdk_width=NULL,z_ui_gdk_height=NULL,z_ui_gdk_stride=NULL,z_ui_gdk_channels=NULL,z_ui_gdk_alpha=NULL;static ZUI_GDK_PIXELS z_ui_gdk_pixels=NULL;static ZUI_GOBJECT_UNREF z_ui_gobject_unref=NULL;
#endif
static void z_ui_load_image_helpers(void){
    if(!z_ui_image_library){
#if defined(_WIN32)
        const char*names[]={"SDL3_image.dll","libSDL3_image.dll",NULL};
#elif defined(__APPLE__)
        const char*names[]={"@executable_path/../Frameworks/libSDL3_image.0.dylib","libSDL3_image.0.dylib","libSDL3_image.dylib",NULL};
#else
        const char*names[]={"libSDL3_image.so.0","libSDL3_image.so",NULL};
#endif
        for(size_t i=0;names[i];i++){z_ui_image_library=z_desktop_library_open(names[i]);if(z_ui_image_library)break;}if(z_ui_image_library)z_ui_img_load=(ZUI_IMG_Load)z_desktop_library_symbol(z_ui_image_library,"IMG_Load");
    }
#if defined(__linux__)
    if(!z_ui_pixbuf_library){z_ui_pixbuf_library=z_desktop_library_open("libgdk_pixbuf-2.0.so.0");z_ui_gobject_library=z_desktop_library_open("libgobject-2.0.so.0");if(z_ui_pixbuf_library&&z_ui_gobject_library){z_ui_gdk_new=(ZUI_GDK_NEW)z_desktop_library_symbol(z_ui_pixbuf_library,"gdk_pixbuf_new_from_file");z_ui_gdk_width=(ZUI_GDK_INT)z_desktop_library_symbol(z_ui_pixbuf_library,"gdk_pixbuf_get_width");z_ui_gdk_height=(ZUI_GDK_INT)z_desktop_library_symbol(z_ui_pixbuf_library,"gdk_pixbuf_get_height");z_ui_gdk_stride=(ZUI_GDK_INT)z_desktop_library_symbol(z_ui_pixbuf_library,"gdk_pixbuf_get_rowstride");z_ui_gdk_channels=(ZUI_GDK_INT)z_desktop_library_symbol(z_ui_pixbuf_library,"gdk_pixbuf_get_n_channels");z_ui_gdk_alpha=(ZUI_GDK_INT)z_desktop_library_symbol(z_ui_pixbuf_library,"gdk_pixbuf_get_has_alpha");z_ui_gdk_pixels=(ZUI_GDK_PIXELS)z_desktop_library_symbol(z_ui_pixbuf_library,"gdk_pixbuf_get_pixels");z_ui_gobject_unref=(ZUI_GOBJECT_UNREF)z_desktop_library_symbol(z_ui_gobject_library,"g_object_unref");}}
#endif
}
static SDL_Surface*z_ui_load_image_surface(const char*path,unsigned char**owned){
    *owned=NULL;z_ui_load_image_helpers();if(z_ui_img_load){SDL_Surface*s=z_ui_img_load(path);if(s)return s;}
#if defined(__linux__)
    if(z_ui_gdk_new&&z_ui_gdk_width&&z_ui_gdk_height&&z_ui_gdk_stride&&z_ui_gdk_channels&&z_ui_gdk_pixels&&z_ui_gobject_unref&&z_desktop_sdl.CreateSurfaceFrom){ZUIGError*err=NULL;ZUIGdkPixbuf*pix=z_ui_gdk_new(path,&err);if(pix){int w=z_ui_gdk_width(pix),h=z_ui_gdk_height(pix),stride=z_ui_gdk_stride(pix),channels=z_ui_gdk_channels(pix),alpha=z_ui_gdk_alpha?z_ui_gdk_alpha(pix):channels==4;unsigned char*src=z_ui_gdk_pixels(pix),*rgba=(unsigned char*)malloc((size_t)w*(size_t)h*4u);if(rgba){for(int y=0;y<h;y++)for(int x=0;x<w;x++){unsigned char*q=rgba+((size_t)y*(size_t)w+(size_t)x)*4u,*v=src+(size_t)y*(size_t)stride+(size_t)x*(size_t)channels;q[0]=v[0];q[1]=v[1];q[2]=v[2];q[3]=alpha&&channels>3?v[3]:255;}SDL_Surface*s=z_desktop_sdl.CreateSurfaceFrom(w,h,Z_UI_PIXELFORMAT_RGBA32,rgba,w*4);if(s){*owned=rgba;z_ui_gobject_unref(pix);return s;}free(rgba);}z_ui_gobject_unref(pix);}}
#endif
    return z_desktop_sdl_load_bmp(path);
}
static bool z_ui_render_image_file(SDL_Renderer*r,const char*path,ZUIRectNative bounds,const char*fit){if(!path||!*path||!z_desktop_sdl.CreateTextureFromSurface||!z_desktop_sdl.RenderTexture||!z_desktop_sdl.DestroyTexture||!z_desktop_sdl.DestroySurface)return false;int64_t modified=0,file_size=0;z_ui_file_signature(path,&modified,&file_size);ZUIImageTextureCacheNative*cached=z_ui_image_cache_find(r,path,modified,file_size);SDL_Texture*t=cached?cached->texture:NULL;float tw=cached?cached->width:0,th=cached?cached->height:0;if(t==NULL){unsigned char*owned=NULL;SDL_Surface*s=z_ui_load_image_surface(path,&owned);if(!s)return false;t=z_desktop_sdl.CreateTextureFromSurface(r,s);z_desktop_sdl.DestroySurface(s);if(owned)free(owned);if(!t)return false;if(z_desktop_sdl.GetTextureSize)z_desktop_sdl.GetTextureSize(t,&tw,&th);ZUIImageTextureCacheNative*stored=z_ui_image_cache_store(r,path,modified,file_size,t,tw,th);if(stored==NULL){ZDesktopSDLFRect fallback={(float)bounds.x,(float)bounds.y,(float)bounds.width,(float)bounds.height};bool ok=z_desktop_sdl.RenderTexture(r,t,NULL,&fallback);z_ui_destroy_texture(t);return ok;}}ZDesktopSDLFRect d={(float)bounds.x,(float)bounds.y,(float)bounds.width,(float)bounds.height},src={0,0,tw,th};ZDesktopSDLFRect*source=NULL;if(fit&&strcmp(fit,"contain")==0&&tw>0&&th>0){float scale=fminf((float)bounds.width/tw,(float)bounds.height/th);d.w=tw*scale;d.h=th*scale;d.x=(float)bounds.x+((float)bounds.width-d.w)/2;d.y=(float)bounds.y+((float)bounds.height-d.h)/2;}else if(fit&&strcmp(fit,"cover")==0&&tw>0&&th>0&&bounds.width>0&&bounds.height>0){float target=(float)bounds.width/(float)bounds.height,actual=tw/th;if(actual>target){src.w=th*target;src.x=(tw-src.w)/2;}else{src.h=tw/target;src.y=(th-src.h)/2;}source=&src;}return z_desktop_sdl.RenderTexture(r,t,source,&d);}
static void z_ui_blur_pixels(unsigned char*pixels,int w,int h,int pitch,int radius){if(!pixels||w<2||h<2||radius<1)return;size_t bytes=(size_t)pitch*(size_t)h;unsigned char*src=(unsigned char*)malloc(bytes),*tmp=(unsigned char*)malloc(bytes);if(!src||!tmp){free(src);free(tmp);return;}memcpy(src,pixels,bytes);for(int y=0;y<h;y++)for(int x=0;x<w;x++){int count=0,sum[4]={0,0,0,0};for(int k=-radius;k<=radius;k++){int sx=x+k;if(sx<0)sx=0;if(sx>=w)sx=w-1;unsigned char*q=src+(size_t)y*(size_t)pitch+(size_t)sx*4u;for(int c=0;c<4;c++)sum[c]+=q[c];count++;}unsigned char*d=tmp+(size_t)y*(size_t)pitch+(size_t)x*4u;for(int c=0;c<4;c++)d[c]=(unsigned char)(sum[c]/count);}for(int y=0;y<h;y++)for(int x=0;x<w;x++){int count=0,sum[4]={0,0,0,0};for(int k=-radius;k<=radius;k++){int sy=y+k;if(sy<0)sy=0;if(sy>=h)sy=h-1;unsigned char*q=tmp+(size_t)sy*(size_t)pitch+(size_t)x*4u;for(int c=0;c<4;c++)sum[c]+=q[c];count++;}unsigned char*d=pixels+(size_t)y*(size_t)pitch+(size_t)x*4u;for(int c=0;c<4;c++)d[c]=(unsigned char)(sum[c]/count);}free(src);free(tmp);}
static bool z_ui_blur_backdrop(SDL_Renderer*r,int width,int height,int radius,const void*owner){if(radius<1||!z_desktop_sdl.RenderReadPixels||!z_desktop_sdl.ConvertSurface||!z_desktop_sdl.CreateTextureFromSurface||!z_desktop_sdl.RenderTexture||!z_desktop_sdl.DestroyTexture||!z_desktop_sdl.DestroySurface)return false;ZUIBackdropCacheNative*cached=z_ui_backdrop_cache_find(r,owner,width,height,radius);if(cached&&cached->texture){ZDesktopSDLFRect d={0,0,(float)width,(float)height};return z_desktop_sdl.RenderTexture(r,cached->texture,NULL,&d);}z_ui_backdrop_cache_clear_renderer(r);SDL_Surface*raw=z_desktop_sdl.RenderReadPixels(r,NULL);if(!raw)return false;SDL_Surface*rgba=z_desktop_sdl.ConvertSurface(raw,Z_UI_PIXELFORMAT_RGBA32);z_desktop_sdl.DestroySurface(raw);if(!rgba)return false;ZUISurfaceView*v=(ZUISurfaceView*)rgba;if(v->pixels&&v->w>0&&v->h>0)z_ui_blur_pixels((unsigned char*)v->pixels,v->w,v->h,v->pitch,radius);SDL_Texture*t=z_desktop_sdl.CreateTextureFromSurface(r,rgba);z_desktop_sdl.DestroySurface(rgba);if(!t)return false;ZUIBackdropCacheNative*entry=(ZUIBackdropCacheNative*)calloc(1,sizeof(*entry));if(entry){entry->renderer=r;entry->owner=owner;entry->width=width;entry->height=height;entry->radius=radius;entry->texture=t;entry->next=z_ui_backdrops;z_ui_backdrops=entry;}ZDesktopSDLFRect d={0,0,(float)width,(float)height};bool ok=z_desktop_sdl.RenderTexture(r,t,NULL,&d);if(entry==NULL)z_ui_destroy_texture(t);return ok;}

static char*z_ui_utf8_slice(const char*text,size_t start,size_t end);
static void z_ui_text_selection(ZUINodeNative*n,int64_t*caret,int64_t*anchor,int64_t*start,int64_t*end);
static double z_ui_text_width_range(const char*text,int64_t start,int64_t end,ZValue props,ZUIThemeNative*theme);
static int64_t z_ui_text_visible_start(ZUINodeNative*n,int64_t caret,ZUIThemeNative*theme);
static void z_ui_render_node(SDL_Renderer*r,ZUINodeNative*n,ZUIContextNative*ctx,bool has_clip,ZUIRectNative clip){z_ui_set_clip(r,has_clip,clip);
    if(n==NULL||!n->visible)return;
    const char*surface=z_ui_theme_string(ctx->theme,"surface","#ffffff");
    const char*text=z_ui_theme_string(ctx->theme,"text","#172033");
    const char*border=z_ui_theme_string(ctx->theme,"border","#cfd6e2");
    const char*primary=z_ui_theme_string(ctx->theme,"primary","#3867e8");
    const char*bg=z_ui_prop_string(n->props,"background","transparent");
    const char*fg=z_ui_prop_string(n->props,"textColor",text);
    ZUIRectNative b=n->bounds;
    if(strcmp(n->kind,"text")==0){z_ui_text(r,b.x+2,b.y+4,z_ui_node_text(n),fg,n->props,ctx->theme);}
    else if(strcmp(n->kind,"button")==0){
        if(strcmp(bg,"transparent")==0)bg=primary;
        z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);
        ZUIRectNative local_clip={b.x+1,b.y+1,fmax(0,b.width-2),fmax(0,b.height-2)};
        if(z_ui_set_local_clip(r,has_clip,clip,local_clip)){
            const char*button_color=z_ui_prop_string(n->props,"textColor",z_ui_theme_string(ctx->theme,"primaryText","#ffffff"));
            if(!z_ui_render_control_icon(r,b,z_ui_prop_string(n->props,"icon",""),button_color))z_ui_text_aligned(r,b,10,z_ui_node_text(n),button_color,n->props,ctx->theme,"center");
        }
        z_ui_set_clip(r,has_clip,clip);
    }
    else if(strcmp(n->kind,"input")==0||strcmp(n->kind,"textarea")==0||strcmp(n->kind,"select")==0){
        bool focused=ctx->focus_id&&strcmp(ctx->focus_id,n->id)==0;
        if(strcmp(bg,"transparent")==0)bg=surface;
        if(focused)bg=z_ui_prop_string(n->props,"focusBackground",z_ui_theme_string(ctx->theme,"surfaceAlt","#eef2f7"));
        z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);
        ZUIRectNative local_clip={b.x+1,b.y+1,fmax(0,b.width-2),fmax(0,b.height-2)};
        if(z_ui_set_local_clip(r,has_clip,clip,local_clip)){
            const char*actual=z_ui_prop_string(n->props,"value","");
            const char*label=*actual?actual:z_ui_prop_string(n->props,"placeholder","");
            if(strcmp(n->kind,"select")==0){
                ZUIRectNative text_bounds=b;text_bounds.width=fmax(0,text_bounds.width-30);
                z_ui_text_aligned(r,text_bounds,8,label,*actual?fg:"#87909f",n->props,ctx->theme,"left");
                double cx=b.x+b.width-15,cy=b.y+b.height/2;z_ui_draw_color(r,fg);
                z_desktop_sdl.RenderLine(r,(float)(cx-4),(float)(cy-2),(float)cx,(float)(cy+2));
                z_desktop_sdl.RenderLine(r,(float)cx,(float)(cy+2),(float)(cx+4),(float)(cy-2));
            }else if(!focused&&!*actual){
                z_ui_text_aligned(r,b,8,label,"#87909f",n->props,ctx->theme,"left");
            }else{
                int64_t caret=0,anchor=0,sel_start=0,sel_end=0;z_ui_text_selection(n,&caret,&anchor,&sel_start,&sel_end);
                int64_t view_start=z_ui_text_visible_start(n,caret,ctx->theme),length=(int64_t)z_ui_utf8_count(actual),view_end=view_start;
                while(view_end<length&&z_ui_text_width_range(actual,view_start,view_end+1,n->props,ctx->theme)<=fmax(0,b.width-16))view_end++;
                char*visible=z_ui_utf8_slice(actual,(size_t)view_start,(size_t)view_end);double tw=0,th=0;
                z_ui_measure_text_native(visible,n->props,ctx->theme,0,&tw,&th);double tx=b.x+8,ty=b.y+fmax(0,(b.height-th)/2);
                if(focused&&sel_start!=sel_end){int64_t a=sel_start>view_start?sel_start:view_start,e=sel_end<view_end?sel_end:view_end;if(a<e){double prefix=z_ui_text_width_range(actual,view_start,a,n->props,ctx->theme),selected=z_ui_text_width_range(actual,a,e,n->props,ctx->theme);z_ui_fill(r,(ZUIRectNative){tx+prefix,ty-2,selected,fmax(th+4,18)},z_ui_prop_string(n->props,"selectionBackground","#3867e855"));}}
                z_ui_text(r,tx,ty,visible,fg,n->props,ctx->theme);
                if(focused){int64_t visible_caret=caret<view_start?view_start:(caret>view_end?view_end:caret);double prefix=z_ui_text_width_range(actual,view_start,visible_caret,n->props,ctx->theme),caret_h=fmin(b.height-12,fmax(14,th)),caret_y=b.y+(b.height-caret_h)/2;z_ui_draw_color(r,z_ui_prop_string(n->props,"caretColor",z_ui_theme_string(ctx->theme,"primary","#3867e8")));z_desktop_sdl.RenderLine(r,(float)(tx+prefix),(float)caret_y,(float)(tx+prefix),(float)(caret_y+caret_h));}
            }
        }
        z_ui_set_clip(r,has_clip,clip);
    }
    else if(strcmp(n->kind,"checkbox")==0||strcmp(n->kind,"radio")==0){ZUIRectNative box={b.x+4,b.y+(b.height-16)/2,16,16};z_ui_fill(r,box,surface);z_ui_stroke(r,box,border);if(z_ui_prop_bool(n->props,"checked",false))z_ui_fill(r,(ZUIRectNative){box.x+4,box.y+4,8,8},primary);ZUIRectNative label_bounds={b.x+20,b.y,b.width-20,b.height};z_ui_text_centered(r,label_bounds,8,z_ui_node_text(n),fg,n->props,ctx->theme);}
    else if(strcmp(n->kind,"progress")==0){z_ui_fill(r,b,z_ui_prop_string(n->props,"background",z_ui_theme_string(ctx->theme,"surfaceAlt","#eef2f7")));double max=z_ui_prop_number(n->props,"max",100),value=z_ui_prop_number(n->props,"value",0);if(max<=0)max=100;if(value<0)value=0;if(value>max)value=max;ZUIRectNative fill=b;fill.width=b.width*value/max;z_ui_fill(r,fill,z_ui_prop_string(n->props,"fill",primary));}
    else if(strcmp(n->kind,"list")==0||strcmp(n->kind,"tree")==0){z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);ZValue items=z_desktop_dict_get(n->props,"items");double row_height=z_ui_prop_number(n->props,"itemHeight",28);if(items.tag==ZV_ARRAY){for(size_t i=0;i<items.as.array->len;i++){double y=b.y+6+(double)i*row_height;if(y>b.y+b.height)break;const char*label=z_ui_value_text(items.as.array->items[i]);if(strcmp(n->kind,"tree")==0){char*line=(char*)z_alloc(strlen(label)+3);sprintf(line,"- %s",label);z_ui_text(r,b.x+8,y,line,fg,n->props,ctx->theme);}else z_ui_text(r,b.x+8,y,label,fg,n->props,ctx->theme);}}}
    else if(strcmp(n->kind,"tabs")==0){z_ui_fill(r,b,bg);ZValue tabs=z_desktop_dict_get(n->props,"tabs");int64_t selected=(int64_t)z_ui_prop_number(n->props,"selectedIndex",0);double x=b.x;if(tabs.tag==ZV_ARRAY){for(size_t i=0;i<tabs.as.array->len;i++){const char*label=z_ui_value_text(tabs.as.array->items[i]);double tw=0,th=0;z_ui_measure_text_native(label,n->props,ctx->theme,0,&tw,&th);double width=fmax(72,tw+24);ZUIRectNative tab={x,b.y,width,b.height};if((int64_t)i==selected)z_ui_fill(r,tab,z_ui_prop_string(n->props,"selectedBackground",surface));z_ui_stroke(r,tab,border);z_ui_text_centered(r,tab,10,label,fg,n->props,ctx->theme);x+=width;}}}
    else if(strcmp(n->kind,"menu")==0){z_ui_fill(r,b,bg);if(z_ui_prop_bool(n->props,"border",true))z_ui_stroke(r,b,border);const char*label=z_ui_node_text(n);if(*label!='\0')z_ui_text_centered(r,b,10,label,fg,n->props,ctx->theme);}
    else if(strcmp(n->kind,"table")==0){z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);ZValue columns=z_desktop_dict_get(n->props,"columns"),rows=z_desktop_dict_get(n->props,"rows");double row_height=z_ui_prop_number(n->props,"rowHeight",30);size_t count=columns.tag==ZV_ARRAY?columns.as.array->len:1;if(count<1)count=1;double column_width=b.width/(double)count;if(columns.tag==ZV_ARRAY){for(size_t i=0;i<columns.as.array->len;i++){double x=b.x+(double)i*column_width;z_ui_text(r,x+6,b.y+6,z_ui_value_text(columns.as.array->items[i]),fg,n->props,ctx->theme);if(i>0){z_ui_draw_color(r,border);z_desktop_sdl.RenderLine(r,(float)x,(float)b.y,(float)x,(float)(b.y+b.height));}}}z_ui_draw_color(r,border);z_desktop_sdl.RenderLine(r,(float)b.x,(float)(b.y+row_height),(float)(b.x+b.width),(float)(b.y+row_height));if(rows.tag==ZV_ARRAY){for(size_t i=0;i<rows.as.array->len;i++){double y=b.y+row_height+(double)i*row_height;if(y+row_height>b.y+b.height+.5)break;ZValue row_value=rows.as.array->items[i];for(size_t column=0;column<count;column++){const char*cell="";if(row_value.tag==ZV_ARRAY&&column<row_value.as.array->len)cell=z_ui_value_text(row_value.as.array->items[column]);else if(column==0)cell=z_ui_value_text(row_value);ZUIRectNative cell_bounds={b.x+(double)column*column_width,y,column_width,row_height};z_ui_text_aligned(r,cell_bounds,6,cell,fg,n->props,ctx->theme,"left");}z_ui_draw_color(r,border);z_desktop_sdl.RenderLine(r,(float)b.x,(float)(y+row_height),(float)(b.x+b.width),(float)(y+row_height));}}}
    else if(strcmp(n->kind,"image")==0){z_ui_render_image_file(r,z_ui_prop_string(n->props,"path",""),b,z_ui_prop_string(n->props,"fit","contain"));}
    else if(strcmp(n->kind,"modal")==0){ZUIRectNative viewport={0,0,b.x*2+b.width,b.y*2+b.height};int blur=(int)z_ui_prop_number(n->props,"backdropBlur",6);if(blur>0)z_ui_blur_backdrop(r,(int)viewport.width,(int)viewport.height,blur,n);z_ui_fill(r,viewport,z_ui_prop_string(n->props,"overlay","#00000055"));z_ui_fill(r,(ZUIRectNative){b.x+8,b.y+10,b.width,b.height},z_ui_prop_string(n->props,"modalShadow","#00000040"));z_ui_fill(r,b,strcmp(bg,"transparent")==0?surface:bg);if(z_ui_prop_bool(n->props,"border",false))z_ui_stroke(r,b,border);}
    else if(strcmp(n->kind,"tooltip")==0){z_ui_fill(r,b,strcmp(bg,"transparent")==0?"#202633":bg);z_ui_text(r,b.x+6,b.y+5,z_ui_node_text(n),z_ui_prop_string(n->props,"textColor","#ffffff"),n->props,ctx->theme);}
    else if(strcmp(n->kind,"canvas")==0){z_ui_fill(r,b,bg);ZValue commands=z_desktop_dict_get(n->props,"commands");if(commands.tag==ZV_ARRAY){for(size_t i=0;i<commands.as.array->len;i++){ZValue command=commands.as.array->items[i];const char*kind=z_ui_prop_string(command,"kind","");ZValue values=z_desktop_dict_get(command,"values");const char*color=z_ui_prop_string(values,"color",text);double x=b.x+z_ui_prop_number(values,"x",0),y=b.y+z_ui_prop_number(values,"y",0);if(strcmp(kind,"fillRect")==0)z_ui_fill(r,(ZUIRectNative){x,y,z_ui_prop_number(values,"width",10),z_ui_prop_number(values,"height",10)},color);else if(strcmp(kind,"rect")==0)z_ui_stroke(r,(ZUIRectNative){x,y,z_ui_prop_number(values,"width",10),z_ui_prop_number(values,"height",10)},color);else if(strcmp(kind,"line")==0){z_ui_draw_color(r,color);z_desktop_sdl.RenderLine(r,(float)x,(float)y,(float)(b.x+z_ui_prop_number(values,"x2",0)),(float)(b.y+z_ui_prop_number(values,"y2",0)));}else if(strcmp(kind,"text")==0)z_ui_text(r,x,y,z_ui_prop_string(values,"text",""),color,values,ctx->theme);else if(strcmp(kind,"pieChart")==0)z_ui_render_pie_chart(r,b,values,ctx->theme);else if(strcmp(kind,"barChart")==0)z_ui_render_bar_chart(r,b,values,ctx->theme);else if(strcmp(kind,"lineChart")==0)z_ui_render_line_chart(r,b,values,ctx->theme);}}}
    else{z_ui_fill(r,b,bg);if(z_ui_prop_bool(n->props,"border",false))z_ui_stroke(r,b,border);}
    if(ctx->focus_id&&strcmp(ctx->focus_id,n->id)==0){double inset=fmax(1,z_ui_prop_number(n->props,"focusInset",1));z_ui_stroke(r,(ZUIRectNative){b.x+inset,b.y+inset,fmax(0,b.width-inset*2),fmax(0,b.height-inset*2)},z_ui_theme_string(ctx->theme,"focus","#6e95ff"));}
    bool child_has_clip=has_clip;ZUIRectNative child_clip=clip;if(z_ui_clips_children(n)){if(has_clip){if(!z_ui_rect_intersection(clip,n->content_bounds,&child_clip))return;}else child_clip=n->content_bounds;child_has_clip=true;}
    for(size_t i=0;i<n->children.as.array->len;i++)z_ui_render_node(r,z_ui_expect_node(n->children.as.array->items[i]),ctx,child_has_clip,child_clip);
    /* A scrollbar with scrollbarAvoidContent=true lives in the reserved gutter
     * beside content_bounds. Restore the inherited viewport before painting it;
     * keeping the child-content clip here hides the thumb while still reserving
     * its width, which leaves an unexplained empty strip on the right. */
    z_ui_set_clip(r,has_clip,clip);
    if(z_ui_scrollable_y(n)&&n->content_height>n->content_bounds.height+0.5&&n->content_bounds.height>0){double bar=fmax(4,z_ui_prop_number(n->props,"scrollbarWidth",8)),gutter=fmax(0,z_ui_prop_number(n->props,"scrollbarGutter",4));double track_x=n->content_bounds.x+n->content_bounds.width+fmax(2,gutter);if(z_ui_prop_bool(n->props,"scrollbarOverlay",false)&&!z_ui_prop_bool(n->props,"scrollbarAvoidContent",false))track_x=n->content_bounds.x+n->content_bounds.width-bar-gutter;ZUIRectNative track={track_x,n->content_bounds.y,bar,n->content_bounds.height};z_ui_fill(r,track,z_ui_prop_string(n->props,"scrollbarTrack","transparent"));double thumb_h=fmax(24,track.height*n->content_bounds.height/n->content_height);if(thumb_h>track.height)thumb_h=track.height;double max_offset=n->content_height-n->content_bounds.height,thumb_y=track.y;if(max_offset>0)thumb_y+=(track.height-thumb_h)*n->scroll_offset_y/max_offset;z_ui_fill(r,(ZUIRectNative){track.x,thumb_y,track.width,thumb_h},z_ui_prop_string(n->props,"scrollbarThumb","#9aa7ba"));}
    z_ui_set_clip(r,has_clip,clip);
}
#endif
static ZUINodeNative*z_ui_topmost_modal(ZUINodeNative*n);
static void z_ui_render_context(ZUIContextNative*ctx){if(ctx==NULL||ctx->closed||ctx->window==NULL||!ctx->window->open)return;z_ui_layout(ctx->root,(ZUIRectNative){0,0,(double)ctx->window->width,(double)ctx->window->height},ctx->theme);
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(!ctx->app->headless){if(ctx->window->renderer==NULL){ctx->window->renderer=z_desktop_sdl.CreateRenderer((SDL_Window*)ctx->window->handle,NULL);if(ctx->window->renderer==NULL)z_fatal("uiRender could not create SDL3 renderer: %s",z_desktop_sdl.GetError());if(z_desktop_sdl.SetRenderDrawBlendMode)z_desktop_sdl.SetRenderDrawBlendMode((SDL_Renderer*)ctx->window->renderer,1u);}SDL_Renderer*r=(SDL_Renderer*)ctx->window->renderer;ZUINodeNative*active_modal=z_ui_topmost_modal(ctx->root);if(active_modal==NULL)z_ui_backdrop_cache_clear_renderer(r);z_ui_draw_color(r,z_ui_theme_string(ctx->theme,"background","#f5f7fb"));z_desktop_sdl.RenderClear(r);z_ui_render_node(r,ctx->root,ctx,false,(ZUIRectNative){0,0,0,0});z_ui_set_clip(r,false,(ZUIRectNative){0,0,0,0});ZUINodeNative*popup_root=z_ui_topmost_modal(ctx->root);if(!popup_root)popup_root=ctx->root;z_ui_render_open_selects(r,popup_root,ctx,(ZUIRectNative){0,0,(double)ctx->window->width,(double)ctx->window->height});z_desktop_sdl.RenderPresent(r);}
#endif
    ctx->dirty=false;}
static void z_ui_render_all(ZDesktopAppNative*app){if(app==NULL)return;for(ZUIContextNative*c=app->ui_contexts;c;c=c->next)if(c->dirty)z_ui_render_context(c);}
static bool z_ui_any_dirty(ZDesktopAppNative*app){if(app==NULL)return false;for(ZUIContextNative*c=app->ui_contexts;c;c=c->next)if(c->dirty&&!c->closed)return true;return false;}
static void z_ui_mark_dirty(ZUIContextNative*ctx){if(ctx==NULL||ctx->closed)return;bool was_dirty=ctx->dirty;ctx->dirty=true;if(!was_dirty)z_desktop_wake_app(ctx->app);}
static bool z_ui_set_hover(ZUIContextNative*ctx,const char*id){if(ctx==NULL)return false;if(ctx->hover_id==NULL&&id==NULL)return false;if(ctx->hover_id!=NULL&&id!=NULL&&strcmp(ctx->hover_id,id)==0)return false;ctx->hover_id=id?z_strdup(id):NULL;return true;}
static ZUINodeNative*z_ui_hit_clipped(ZUINodeNative*n,double x,double y,bool has_clip,ZUIRectNative clip){if(n==NULL||!n->visible)return NULL;if(has_clip&&!z_ui_point_in_rect(x,y,clip))return NULL;bool child_has_clip=has_clip;ZUIRectNative child_clip=clip;if(z_ui_clips_children(n)){if(has_clip){if(!z_ui_rect_intersection(clip,n->content_bounds,&child_clip))return NULL;}else child_clip=n->content_bounds;child_has_clip=true;}for(size_t i=n->children.as.array->len;i>0;i--){ZUINodeNative*h=z_ui_hit_clipped(z_ui_expect_node(n->children.as.array->items[i-1]),x,y,child_has_clip,child_clip);if(h)return h;}return z_ui_point_in_rect(x,y,n->bounds)?n:NULL;}
static ZUINodeNative*z_ui_topmost_modal(ZUINodeNative*n){if(n==NULL||!n->visible)return NULL;for(size_t i=n->children.as.array->len;i>0;i--){ZUINodeNative*m=z_ui_topmost_modal(z_ui_expect_node(n->children.as.array->items[i-1]));if(m)return m;}return strcmp(n->kind,"modal")==0?n:NULL;}
static ZUINodeNative*z_ui_hit(ZUINodeNative*n,double x,double y){ZUINodeNative*modal=z_ui_topmost_modal(n);if(modal){ZUINodeNative*hit=z_ui_hit_clipped(modal,x,y,false,(ZUIRectNative){0,0,0,0});return hit?hit:modal;}return z_ui_hit_clipped(n,x,y,false,(ZUIRectNative){0,0,0,0});}
static bool z_ui_effectively_visible(ZUINodeNative*n){for(ZUINodeNative*current=n;current;current=current->parent)if(!current->visible)return false;return n!=NULL;}
static bool z_ui_focusable(ZUINodeNative*n){if(!n||!z_ui_effectively_visible(n)||!n->enabled||z_ui_prop_number(n->props,"tabIndex",0)<0)return false;return strcmp(n->kind,"button")==0||strcmp(n->kind,"input")==0||strcmp(n->kind,"textarea")==0||strcmp(n->kind,"select")==0||strcmp(n->kind,"checkbox")==0||strcmp(n->kind,"radio")==0||strcmp(n->kind,"tabs")==0||strcmp(n->kind,"menu")==0||z_ui_prop_bool(n->props,"focusable",false);}
static void z_ui_call_prop(ZUINodeNative*n,const char*name,ZValue event){ZValue h=z_desktop_dict_get(n->props,name);if(h.tag==ZV_FUNCTION||h.tag==ZV_BUILTIN||h.tag==ZV_BOUND_METHOD)(void)z_call(h,&event,1);}
static ZUINodeNative*z_ui_find_node(ZUINodeNative*n,const char*id){if(n==NULL)return NULL;if(strcmp(n->id,id)==0)return n;for(size_t i=0;i<n->children.as.array->len;i++){ZUINodeNative*r=z_ui_find_node(z_ui_expect_node(n->children.as.array->items[i]),id);if(r)return r;}return NULL;}
static void z_ui_apply_property(ZUINodeNative*n,const char*property,ZValue value){z_ui_dict_set(n->props,property,value);if(strcmp(property,"visible")==0&&value.tag==ZV_BOOL)n->visible=value.as.b;if(strcmp(property,"disabled")==0&&value.tag==ZV_BOOL)n->enabled=!value.as.b;}
static ZValue z_ui_state_set_value(ZUIStateNative*s,ZValue value){s->value=value;for(ZUIBindingNative*b=s->bindings;b;b=b->next){z_ui_apply_property(b->node,b->property,value);if(b->node->context)z_ui_mark_dirty(b->node->context);}for(ZUISubscriberNative*h=s->subscribers;h;h=h->next)(void)z_call(h->handler,&value,1);return value;}
static ZUIStateNative*z_ui_bound_state(ZUINodeNative*n,const char*property){for(ZUIReverseBindingNative*b=n?n->bindings:NULL;b;b=b->next)if(strcmp(b->property,property)==0)return b->state;return NULL;}
static void z_ui_update_bound(ZUINodeNative*n,const char*property,ZValue value){ZUIStateNative*s=z_ui_bound_state(n,property);if(s)z_ui_state_set_value(s,value);}
static void z_ui_collect_focusable_nodes(ZUINodeNative*n,ZUINodeNative**items,size_t*count,size_t cap){if(!n||*count>=cap)return;if(z_ui_focusable(n))items[(*count)++]=n;for(size_t i=0;i<n->children.as.array->len;i++)z_ui_collect_focusable_nodes(z_ui_expect_node(n->children.as.array->items[i]),items,count,cap);}
static size_t z_ui_utf8_byte_offset(const char*text,size_t index){size_t count=0,offset=0;if(!text)return 0;while(text[offset]&&count<index){offset++;while(text[offset]&&(((unsigned char)text[offset]&0xC0u)==0x80u))offset++;count++;}return offset;}
static char*z_ui_utf8_slice(const char*text,size_t start,size_t end){if(!text)text="";size_t a=z_ui_utf8_byte_offset(text,start),b=z_ui_utf8_byte_offset(text,end);if(b<a)b=a;char*out=(char*)z_alloc(b-a+1);memcpy(out,text+a,b-a);out[b-a]='\0';return out;}
static int64_t z_ui_text_index(ZValue props,const char*key,int64_t fallback,int64_t length){int64_t value=(int64_t)z_ui_prop_number(props,key,(double)fallback);if(value<0)value=0;if(value>length)value=length;return value;}
static void z_ui_text_selection(ZUINodeNative*n,int64_t*caret,int64_t*anchor,int64_t*start,int64_t*end){const char*text=z_ui_prop_string(n->props,"value","");int64_t length=(int64_t)z_ui_utf8_count(text);*caret=z_ui_text_index(n->props,"caretIndex",length,length);*anchor=z_ui_text_index(n->props,"selectionAnchor",*caret,length);*start=*caret<*anchor?*caret:*anchor;*end=*caret>*anchor?*caret:*anchor;}
static void z_ui_text_set_selection(ZUINodeNative*n,int64_t caret,int64_t anchor){int64_t length=(int64_t)z_ui_utf8_count(z_ui_prop_string(n->props,"value",""));if(caret<0)caret=0;if(caret>length)caret=length;if(anchor<0)anchor=0;if(anchor>length)anchor=length;int64_t start=caret<anchor?caret:anchor,end=caret>anchor?caret:anchor;z_ui_dict_set(n->props,"caretIndex",z_int(caret));z_ui_dict_set(n->props,"selectionAnchor",z_int(anchor));z_ui_dict_set(n->props,"selectionStart",z_int(start));z_ui_dict_set(n->props,"selectionEnd",z_int(end));}
static char*z_ui_text_replace(const char*text,int64_t start,int64_t end,const char*added){if(!text)text="";if(!added)added="";size_t a=z_ui_utf8_byte_offset(text,(size_t)start),b=z_ui_utf8_byte_offset(text,(size_t)end),n=strlen(text),m=strlen(added);char*out=(char*)z_alloc(a+m+(n-b)+1);memcpy(out,text,a);memcpy(out+a,added,m);memcpy(out+a+m,text+b,n-b);out[a+m+n-b]='\0';return out;}
static void z_ui_text_commit(ZUINodeNative*n,const char*text,int64_t caret,ZValue ev){ZValue value=z_string(text?text:"");z_ui_dict_set(n->props,"value",value);z_ui_text_set_selection(n,caret,caret);z_ui_update_bound(n,"value",value);z_ui_call_prop(n,"onInput",ev);z_ui_call_prop(n,"onChange",ev);}
static int64_t z_ui_text_word_left(const char*text,int64_t index){if(!text)return 0;while(index>0){char*part=z_ui_utf8_slice(text,(size_t)index-1,(size_t)index);bool space=isspace((unsigned char)part[0]);if(!space)break;index--;}while(index>0){char*part=z_ui_utf8_slice(text,(size_t)index-1,(size_t)index);bool space=isspace((unsigned char)part[0]);if(space)break;index--;}return index;}
static int64_t z_ui_text_word_right(const char*text,int64_t index){int64_t length=(int64_t)z_ui_utf8_count(text);while(index<length){char*part=z_ui_utf8_slice(text,(size_t)index,(size_t)index+1);if(!isspace((unsigned char)part[0]))break;index++;}while(index<length){char*part=z_ui_utf8_slice(text,(size_t)index,(size_t)index+1);if(isspace((unsigned char)part[0]))break;index++;}return index;}
static double z_ui_text_width_range(const char*text,int64_t start,int64_t end,ZValue props,ZUIThemeNative*theme){char*part=z_ui_utf8_slice(text,(size_t)start,(size_t)end);double w=0,h=0;z_ui_measure_text_native(part,props,theme,0,&w,&h);return w;}
static int64_t z_ui_text_visible_start(ZUINodeNative*n,int64_t caret,ZUIThemeNative*theme){const char*text=z_ui_prop_string(n->props,"value","");double available=fmax(0,n->bounds.width-16);int64_t start=0;while(start<caret&&z_ui_text_width_range(text,start,caret,n->props,theme)>available)start++;return start;}
static int64_t z_ui_text_caret_from_x(ZUIContextNative*ctx,ZUINodeNative*n,double x){const char*text=z_ui_prop_string(n->props,"value","");int64_t caret=0,anchor=0,start_sel=0,end_sel=0;z_ui_text_selection(n,&caret,&anchor,&start_sel,&end_sel);int64_t length=(int64_t)z_ui_utf8_count(text),start=z_ui_text_visible_start(n,caret,ctx->theme);double relative=x-n->bounds.x-8,previous=0;if(relative<=0)return start;for(int64_t i=start;i<length;i++){double width=z_ui_text_width_range(text,start,i+1,n->props,ctx->theme);if(relative<previous+(width-previous)/2)return i;if(relative<width)return i+1;previous=width;if(width>n->bounds.width-16)break;}return length;}
static void z_ui_text_select_word(ZUINodeNative*n,int64_t index){const char*text=z_ui_prop_string(n->props,"value","");int64_t length=(int64_t)z_ui_utf8_count(text);if(length==0){z_ui_text_set_selection(n,0,0);return;}if(index>=length)index=length-1;if(index<0)index=0;int64_t start=index,end=index;while(start>0){char*p=z_ui_utf8_slice(text,(size_t)start-1,(size_t)start);if(isspace((unsigned char)p[0]))break;start--;}while(end<length){char*p=z_ui_utf8_slice(text,(size_t)end,(size_t)end+1);if(isspace((unsigned char)p[0]))break;end++;}z_ui_text_set_selection(n,end,start);}
static char*z_ui_text_selected(ZUINodeNative*n){int64_t caret=0,anchor=0,start=0,end=0;z_ui_text_selection(n,&caret,&anchor,&start,&end);return z_ui_utf8_slice(z_ui_prop_string(n->props,"value",""),(size_t)start,(size_t)end);}
static bool z_ui_text_handle_key(ZUIContextNative*ctx,ZUINodeNative*n,const char*key,const char*shortcut,ZValue ev){const char*text=z_ui_prop_string(n->props,"value","");int64_t caret=0,anchor=0,start=0,end=0;z_ui_text_selection(n,&caret,&anchor,&start,&end);int64_t length=(int64_t)z_ui_utf8_count(text);bool shift=strstr(shortcut,"shift+")!=NULL,ctrl=strstr(shortcut,"ctrl+")!=NULL||strstr(shortcut,"super+")!=NULL;
#define ZUI_MOVE(next_value) do{int64_t zui_next_value=(next_value);z_ui_text_set_selection(n,zui_next_value,shift?anchor:zui_next_value);return true;}while(0)
if(strcasecmp(shortcut,"ctrl+a")==0||strcasecmp(shortcut,"super+a")==0){z_ui_text_set_selection(n,length,0);return true;}
if(strcasecmp(shortcut,"ctrl+c")==0||strcasecmp(shortcut,"super+c")==0){char*selected=z_ui_text_selected(n);if(*selected){if(ctx->app->headless)ctx->app->clipboard=z_strdup(selected);else z_desktop_sdl.SetClipboardText(selected);}return true;}
if(strcasecmp(shortcut,"ctrl+x")==0||strcasecmp(shortcut,"super+x")==0){char*selected=z_ui_text_selected(n);if(*selected){if(ctx->app->headless)ctx->app->clipboard=z_strdup(selected);else z_desktop_sdl.SetClipboardText(selected);char*next=z_ui_text_replace(text,start,end,"");z_ui_text_commit(n,next,start,ev);}return true;}
if(strcasecmp(shortcut,"ctrl+v")==0||strcasecmp(shortcut,"super+v")==0){char*clip=NULL;bool release=false;if(ctx->app->headless)clip=ctx->app->clipboard?ctx->app->clipboard:"";else{clip=z_desktop_sdl.GetClipboardText();release=true;}if(!clip)clip="";char*next=z_ui_text_replace(text,start,end,clip);z_ui_text_commit(n,next,start+(int64_t)z_ui_utf8_count(clip),ev);if(release&&clip)z_desktop_sdl.Free(clip);return true;}
bool left=strcasecmp(key,"left")==0||strcasecmp(key,"arrowleft")==0,right=strcasecmp(key,"right")==0||strcasecmp(key,"arrowright")==0;
if(left){int64_t next=caret;if(!shift&&start!=end)next=start;else if(ctrl)next=z_ui_text_word_left(text,caret);else if(next>0)next--;ZUI_MOVE(next);}
if(right){int64_t next=caret;if(!shift&&start!=end)next=end;else if(ctrl)next=z_ui_text_word_right(text,caret);else if(next<length)next++;ZUI_MOVE(next);}
if(strcasecmp(key,"home")==0)ZUI_MOVE(0);if(strcasecmp(key,"end")==0)ZUI_MOVE(length);
if(strcasecmp(key,"backspace")==0){if(start!=end){char*next=z_ui_text_replace(text,start,end,"");z_ui_text_commit(n,next,start,ev);}else if(caret>0){char*next=z_ui_text_replace(text,caret-1,caret,"");z_ui_text_commit(n,next,caret-1,ev);}return true;}
if(strcasecmp(key,"delete")==0){if(start!=end){char*next=z_ui_text_replace(text,start,end,"");z_ui_text_commit(n,next,start,ev);}else if(caret<length){char*next=z_ui_text_replace(text,caret,caret+1,"");z_ui_text_commit(n,next,caret,ev);}return true;}
#undef ZUI_MOVE
return false;}

static bool z_ui_accepts_text_input(ZUINodeNative*n){return n!=NULL&&n->visible&&n->enabled&&(strcmp(n->kind,"input")==0||strcmp(n->kind,"textarea")==0);}
static void z_ui_sync_text_input(ZUIContextNative*ctx,ZUINodeNative*n){
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(ctx==NULL||ctx->app==NULL||ctx->window==NULL||ctx->window->handle==NULL||ctx->app->headless)return;
    SDL_Window*w=(SDL_Window*)ctx->window->handle;
    if(z_ui_accepts_text_input(n))z_desktop_sdl.StartTextInput(w);else z_desktop_sdl.StopTextInput(w);
#else
    (void)ctx;(void)n;
#endif
}
static void z_ui_dispatch_event(ZUIContextNative*ctx,ZDesktopEventNative*event){
    if(!ctx||ctx->closed||!event)return;
    double x=z_ui_prop_number(event->data,"x",0),y=z_ui_prop_number(event->data,"y",0);
    bool pointer_event=strcmp(event->type,"mouse_motion")==0||strcmp(event->type,"mouse_move")==0||strcmp(event->type,"mouse_down")==0||strcmp(event->type,"mouse_button_down")==0||strcmp(event->type,"mouse_up")==0||strcmp(event->type,"mouse_button_up")==0||strcmp(event->type,"mouse_wheel")==0;
    ZUINodeNative*popup_root=z_ui_topmost_modal(ctx->root);if(!popup_root)popup_root=ctx->root;
    ZUINodeNative*open_select=z_ui_topmost_open_select(popup_root);ZUIRectNative popup={0,0,0,0};double popup_row=0;size_t popup_visible=0;
    if(open_select)popup=z_ui_select_popup_geometry(open_select,(ZUIRectNative){0,0,(double)ctx->window->width,(double)ctx->window->height},&popup_row,&popup_visible);
    ZUINodeNative*target=NULL;
    if(pointer_event){if(open_select&&z_ui_point_in_rect(x,y,popup))target=open_select;else target=z_ui_hit(ctx->root,x,y);if((strcmp(event->type,"mouse_move")==0||strcmp(event->type,"mouse_motion")==0)&&z_ui_prop_number(event->data,"buttons",0)!=0&&ctx->focus_id){ZUINodeNative*focused=z_ui_find_node(ctx->root,ctx->focus_id);if(focused&&z_ui_prop_bool(focused->props,"selecting",false)&&(strcmp(focused->kind,"input")==0||strcmp(focused->kind,"textarea")==0))target=focused;}}
    else if(strcmp(event->type,"key_down")==0||strcmp(event->type,"text_input")==0)target=z_ui_find_node(ctx->root,ctx->focus_id?ctx->focus_id:"");
    if(strcmp(event->type,"mouse_wheel")==0&&open_select&&target==open_select&&z_ui_point_in_rect(x,y,popup)){ZValue options=z_desktop_dict_get(open_select->props,"options");int64_t max_visible=(int64_t)fmax(1,z_ui_prop_number(open_select->props,"maxVisibleOptions",8)),max_offset=options.tag==ZV_ARRAY?(int64_t)options.as.array->len-max_visible:0;if(max_offset<0)max_offset=0;int64_t offset=(int64_t)z_ui_prop_number(open_select->props,"popupOffset",0);double dy=z_ui_prop_number(event->data,"dy",z_ui_prop_number(event->data,"wheelY",0));if(dy>0)offset--;else if(dy<0)offset++;if(offset<0)offset=0;if(offset>max_offset)offset=max_offset;z_ui_dict_set(open_select->props,"popupOffset",z_int(offset));z_ui_mark_dirty(ctx);return;}
    if(strcmp(event->type,"mouse_wheel")==0){ZUINodeNative*scroll=target;while(scroll&&!z_ui_scrollable_y(scroll))scroll=scroll->parent;if(scroll){double dy=z_ui_prop_number(event->data,"dy",z_ui_prop_number(event->data,"wheelY",0)),step=z_ui_prop_number(scroll->props,"scrollStep",48),max_offset=fmax(0,scroll->content_height-scroll->content_bounds.height);scroll->scroll_offset_y=z_ui_clamp(scroll->scroll_offset_y-dy*step,0,max_offset);z_ui_mark_dirty(ctx);}return;}
    const char*shortcut=z_ui_prop_string(event->data,"shortcut","");
    if(strcmp(event->type,"key_down")==0&&(strcasecmp(shortcut,"tab")==0||strcasecmp(shortcut,"shift+tab")==0)){z_ui_close_open_selects(ctx->root,NULL);ZUINodeNative*nodes[2048];size_t count=0;ZUINodeNative*focus_root=z_ui_topmost_modal(ctx->root);if(!focus_root)focus_root=ctx->root;z_ui_collect_focusable_nodes(focus_root,nodes,&count,2048);if(count){size_t current=count;for(size_t i=0;i<count;i++)if(ctx->focus_id&&strcmp(nodes[i]->id,ctx->focus_id)==0){current=i;break;}bool reverse=strcasecmp(shortcut,"shift+tab")==0;size_t next=current==count?(reverse?count-1:0):(reverse?(current+count-1)%count:(current+1)%count);ctx->focus_id=z_strdup(nodes[next]->id);z_ui_sync_text_input(ctx,nodes[next]);z_ui_mark_dirty(ctx);}return;}
    bool down=strcmp(event->type,"mouse_down")==0||strcmp(event->type,"mouse_button_down")==0,up=strcmp(event->type,"mouse_up")==0||strcmp(event->type,"mouse_button_up")==0;
    if(down&&open_select&&target!=open_select){if(z_ui_close_open_selects(ctx->root,NULL))z_ui_mark_dirty(ctx);}
    if(!target||!target->enabled){if(strcmp(event->type,"mouse_move")==0||strcmp(event->type,"mouse_motion")==0){if(z_ui_set_hover(ctx,NULL)){z_desktop_set_system_cursor(0);z_ui_mark_dirty(ctx);}}if(down){ctx->focus_id=NULL;z_ui_sync_text_input(ctx,NULL);z_ui_mark_dirty(ctx);}return;}
    ZValue ev=z_desktop_event_value(event);z_ui_dict_set(ev,"targetId",z_string(target->id));z_ui_dict_set(ev,"targetKind",z_string(target->kind));
    if(down){if(z_ui_focusable(target)){ctx->focus_id=z_strdup(target->id);z_ui_sync_text_input(ctx,target);z_ui_call_prop(target,"onFocus",ev);}else{ctx->focus_id=NULL;z_ui_sync_text_input(ctx,NULL);}if(strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0){int64_t index=z_ui_text_caret_from_x(ctx,target,x);if((int64_t)z_ui_prop_number(event->data,"clicks",0)>=2)z_ui_text_select_word(target,index);else z_ui_text_set_selection(target,index,index);z_ui_dict_set(target->props,"selecting",z_bool(true));}}
    if((strcmp(event->type,"mouse_move")==0||strcmp(event->type,"mouse_motion")==0)){bool hover_changed=z_ui_set_hover(ctx,target->id);if(hover_changed){const char*cursor=z_ui_prop_string(target->props,"cursor",strcmp(target->kind,"button")==0||strcmp(target->kind,"select")==0||strcmp(target->kind,"checkbox")==0||strcmp(target->kind,"radio")==0?"pointer":(strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0?"text":"default"));int cursor_kind=(strcasecmp(cursor,"pointer")==0||strcasecmp(cursor,"hand")==0)?1:((strcasecmp(cursor,"text")==0||strcasecmp(cursor,"ibeam")==0||strcasecmp(cursor,"i-beam")==0)?2:0);z_desktop_set_system_cursor(cursor_kind);z_ui_call_prop(target,"onHover",ev);}if((strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0)&&z_ui_prop_bool(target->props,"selecting",false)&&z_ui_prop_number(event->data,"buttons",0)!=0){int64_t caret=0,anchor=0,start=0,end=0;z_ui_text_selection(target,&caret,&anchor,&start,&end);int64_t next_caret=z_ui_text_caret_from_x(ctx,target,x);if(next_caret!=caret){z_ui_text_set_selection(target,next_caret,anchor);z_ui_mark_dirty(ctx);}return;}if(hover_changed)z_ui_mark_dirty(ctx);return;}
    if(up&&(strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0))z_ui_dict_set(target->props,"selecting",z_bool(false));
    const char*key=z_ui_prop_string(event->data,"key","");
    if(strcmp(event->type,"key_down")==0&&strcmp(target->kind,"select")==0){ZValue options=z_desktop_dict_get(target->props,"options");if(strcasecmp(key,"escape")==0){z_ui_dict_set(target->props,"open",z_bool(false));z_ui_mark_dirty(ctx);return;}if(options.tag==ZV_ARRAY&&options.as.array->len>0&&(strcasecmp(key,"arrowdown")==0||strcasecmp(key,"down")==0||strcasecmp(key,"arrowup")==0||strcasecmp(key,"up")==0)){int64_t index=z_ui_select_current_index(target,options);if(strcasecmp(key,"arrowup")==0||strcasecmp(key,"up")==0)index=(index+(int64_t)options.as.array->len-1)%(int64_t)options.as.array->len;else index=(index+1)%(int64_t)options.as.array->len;z_ui_select_set_value(target,index,ev);z_ui_dict_set(target->props,"open",z_bool(true));z_ui_mark_dirty(ctx);return;}}
    if(up&&strcmp(target->kind,"select")==0&&open_select==target&&z_ui_point_in_rect(x,y,popup)&&popup_row>0){int64_t offset=(int64_t)z_ui_prop_number(target->props,"popupOffset",0),index=offset+(int64_t)((y-popup.y)/popup_row);z_ui_select_set_value(target,index,ev);z_ui_mark_dirty(ctx);return;}
    bool activate=up||(strcmp(event->type,"key_down")==0&&(strcasecmp(key,"enter")==0||strcasecmp(key,"space")==0));
    if(activate){if(strcmp(target->kind,"button")==0||strcmp(target->kind,"menu")==0)z_ui_call_prop(target,"onClick",ev);else if(strcmp(target->kind,"checkbox")==0){bool checked=!z_ui_prop_bool(target->props,"checked",false);ZValue value=z_bool(checked);z_ui_dict_set(target->props,"checked",value);z_ui_update_bound(target,"checked",value);z_ui_call_prop(target,"onChange",ev);}else if(strcmp(target->kind,"radio")==0){ZValue value=z_bool(true);z_ui_dict_set(target->props,"checked",value);z_ui_update_bound(target,"checked",value);if(target->parent){for(size_t i=0;i<target->parent->children.as.array->len;i++){ZUINodeNative*sibling=z_ui_expect_node(target->parent->children.as.array->items[i]);if(sibling!=target&&strcmp(sibling->kind,"radio")==0)z_ui_dict_set(sibling->props,"checked",z_bool(false));}}z_ui_call_prop(target,"onChange",ev);}else if(strcmp(target->kind,"select")==0){bool open=z_ui_prop_bool(target->props,"open",false);z_ui_close_open_selects(ctx->root,target);if(!open){ZValue options=z_desktop_dict_get(target->props,"options");int64_t selected=z_ui_select_current_index(target,options),max_visible=(int64_t)fmax(1,z_ui_prop_number(target->props,"maxVisibleOptions",8)),offset=selected>=max_visible?selected-max_visible+1:0;z_ui_dict_set(target->props,"popupOffset",z_int(offset));}z_ui_dict_set(target->props,"open",z_bool(!open));}else if(strcmp(target->kind,"tabs")==0){ZValue items=z_desktop_dict_get(target->props,"tabs");if(items.tag==ZV_ARRAY&&items.as.array->len>0){int64_t index=(int64_t)z_ui_prop_number(target->props,"selectedIndex",0);index=(index+1)%(int64_t)items.as.array->len;ZValue selected_index=z_int(index),selected_value=z_string(z_ui_value_text(items.as.array->items[index]));z_ui_dict_set(target->props,"selectedIndex",selected_index);z_ui_dict_set(target->props,"value",selected_value);z_ui_update_bound(target,"selectedIndex",selected_index);z_ui_update_bound(target,"value",selected_value);z_ui_call_prop(target,"onChange",ev);}}}
    if(strcmp(event->type,"key_down")==0&&(strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0)){if(z_ui_text_handle_key(ctx,target,key,shortcut,ev)){z_ui_mark_dirty(ctx);return;}}
    if(strcmp(event->type,"text_input")==0&&(strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0)){const char*added=z_ui_prop_string(event->data,"text","");if(*added){int64_t caret=0,anchor=0,start=0,end=0;z_ui_text_selection(target,&caret,&anchor,&start,&end);char*next=z_ui_text_replace(z_ui_prop_string(target->props,"value",""),start,end,added);z_ui_text_commit(target,next,start+(int64_t)z_ui_utf8_count(added),ev);}}
    z_ui_mark_dirty(ctx);
}
static void z_ui_dispatch_from_desktop(ZDesktopAppNative*app,ZDesktopEventNative*event){if(!app||!event)return;for(ZUIContextNative*c=app->ui_contexts;c;c=c->next)if(c->window&&c->window->id==event->window_id)z_ui_dispatch_event(c,event);}
static ZValue z_ui_snapshot_node(ZUINodeNative*n){ZValue pairs[11]={z_pair(z_string("id"),z_string(n->id)),z_pair(z_string("kind"),z_string(n->kind)),z_pair(z_string("x"),z_float(n->bounds.x)),z_pair(z_string("y"),z_float(n->bounds.y)),z_pair(z_string("width"),z_float(n->bounds.width)),z_pair(z_string("height"),z_float(n->bounds.height)),z_pair(z_string("contentX"),z_float(n->content_bounds.x)),z_pair(z_string("contentY"),z_float(n->content_bounds.y)),z_pair(z_string("contentWidth"),z_float(n->content_bounds.width)),z_pair(z_string("contentHeight"),z_float(n->content_height)),z_pair(z_string("scrollOffsetY"),z_float(n->scroll_offset_y))};return z_dict_from(pairs,11);}
static void z_ui_collect_snapshot(ZUINodeNative*n,ZValue*items,size_t*count,size_t cap){if(!n||*count>=cap)return;items[(*count)++]=z_ui_snapshot_node(n);for(size_t i=0;i<n->children.as.array->len;i++)z_ui_collect_snapshot(z_ui_expect_node(n->children.as.array->items[i]),items,count,cap);}
static ZValue z_ui_snapshot(ZUIContextNative*ctx){z_ui_render_context(ctx);ZValue items[2048];size_t count=0;z_ui_collect_snapshot(ctx->root,items,&count,2048);ZValue pairs[4]={z_pair(z_string("width"),z_float(ctx->window->width)),z_pair(z_string("height"),z_float(ctx->window->height)),z_pair(z_string("background"),z_string(z_ui_theme_string(ctx->theme,"background","#f5f7fb"))),z_pair(z_string("items"),z_array_from(items,count))};return z_dict_from(pairs,4);}
static void z_ui_collect_accessibility(ZUINodeNative*n,ZValue*items,size_t*count,size_t cap){if(!n||!n->visible||*count>=cap)return;const char*label=z_ui_prop_string(n->props,"accessibilityLabel",z_ui_prop_string(n->props,"text",z_ui_prop_string(n->props,"label","")));ZValue pairs[5]={z_pair(z_string("id"),z_string(n->id)),z_pair(z_string("role"),z_string(z_ui_prop_string(n->props,"role",n->kind))),z_pair(z_string("label"),z_string(label)),z_pair(z_string("enabled"),z_bool(n->enabled)),z_pair(z_string("focusable"),z_bool(z_ui_focusable(n)))};items[(*count)++]=z_dict_from(pairs,5);for(size_t i=0;i<n->children.as.array->len;i++)z_ui_collect_accessibility(z_ui_expect_node(n->children.as.array->items[i]),items,count,cap);}
static ZValue z_ui_method(ZValue receiver,const char*builtin){ZUIMethodNative*m=(ZUIMethodNative*)z_alloc(sizeof(*m));m->receiver=receiver;m->builtin=builtin;return z_ui_value(ZV_UI_METHOD,ZK_UNKNOWN,m);}
static ZValue z_ui_call_method(ZValue callable,const ZValue*args,size_t argc){ZUIMethodNative*m=(ZUIMethodNative*)callable.as.p;ZValue*full=(ZValue*)z_alloc(sizeof(ZValue)*(argc+1));full[0]=m->receiver;if(argc)memcpy(full+1,args,sizeof(ZValue)*argc);return z_ui_call_builtin(m->builtin,full,argc+1,NULL);}
static ZValue z_ui_get_field(ZValue value,const char*name,bool*handled){*handled=true;if(value.tag==ZV_UI_CONTEXT){if(strcmp(name,"render")==0)return z_ui_method(value,"uiRender");if(strcmp(name,"snapshot")==0)return z_ui_method(value,"uiSnapshot");if(strcmp(name,"setTheme")==0)return z_ui_method(value,"uiSetTheme");if(strcmp(name,"dispatch")==0)return z_ui_method(value,"uiDispatch");if(strcmp(name,"find")==0)return z_ui_method(value,"uiFind");if(strcmp(name,"focus")==0)return z_ui_method(value,"uiFocus");if(strcmp(name,"focusNext")==0)return z_ui_method(value,"uiFocusNext");if(strcmp(name,"accessibility")==0)return z_ui_method(value,"uiAccessibility");if(strcmp(name,"close")==0)return z_ui_method(value,"uiUnmount");}if(value.tag==ZV_UI_NODE){if(strcmp(name,"set")==0)return z_ui_method(value,"uiSet");if(strcmp(name,"get")==0)return z_ui_method(value,"uiGet");if(strcmp(name,"add")==0)return z_ui_method(value,"uiAdd");if(strcmp(name,"remove")==0)return z_ui_method(value,"uiRemove");if(strcmp(name,"command")==0)return z_ui_method(value,"uiCanvasCommand");}if(value.tag==ZV_UI_STATE){if(strcmp(name,"get")==0)return z_ui_method(value,"uiStateGet");if(strcmp(name,"set")==0)return z_ui_method(value,"uiStateSet");if(strcmp(name,"subscribe")==0)return z_ui_method(value,"uiStateSubscribe");}*handled=false;return z_null();}
static ZValue z_ui_call_builtin(const char*name,const ZValue*args,size_t argc,bool*handled){if(handled)*handled=true;
    if(strcmp(name,"uiTheme")==0){z_expect_args(name,argc,1);return z_ui_value(ZV_UI_THEME,ZK_UI_THEME,z_ui_theme_new(args[0]));}
    if(strcmp(name,"uiState")==0){z_expect_args(name,argc,1);ZUIStateNative*s=(ZUIStateNative*)z_alloc(sizeof(*s));memset(s,0,sizeof(*s));s->value=args[0];return z_ui_value(ZV_UI_STATE,ZK_UI_STATE,s);}
    if(strcmp(name,"uiStateGet")==0){z_expect_args(name,argc,1);return z_ui_expect_state(args[0])->value;}
    if(strcmp(name,"uiStateSet")==0){z_expect_args(name,argc,2);return z_ui_state_set_value(z_ui_expect_state(args[0]),args[1]);}
    if(strcmp(name,"uiStateSubscribe")==0){z_expect_args(name,argc,2);ZUIStateNative*s=z_ui_expect_state(args[0]);ZUISubscriberNative*h=(ZUISubscriberNative*)z_alloc(sizeof(*h));h->handler=args[1];h->next=s->subscribers;s->subscribers=h;return args[0];}
    if(strcmp(name,"uiBind")==0){z_expect_args(name,argc,3);ZUINodeNative*n=z_ui_expect_node(args[0]);ZUIStateNative*s=z_ui_expect_state(args[2]);const char*property=z_as_cstring(args[1]);ZUIBindingNative*b=(ZUIBindingNative*)z_alloc(sizeof(*b));b->node=n;b->property=z_strdup(property);b->next=s->bindings;s->bindings=b;ZUIReverseBindingNative*r=(ZUIReverseBindingNative*)z_alloc(sizeof(*r));r->property=z_strdup(property);r->state=s;r->next=n->bindings;n->bindings=r;z_ui_apply_property(n,property,s->value);return args[0];}
    const char*kind=NULL;if(strcmp(name,"uiNode")==0){z_expect_args(name,argc,3);kind=z_as_cstring(args[0]);ZUINodeNative*n=z_ui_node_new(kind,args[1],args[2]);return z_ui_value(ZV_UI_NODE,ZK_UI_NODE,n);}const char*prefix="ui";if(strncmp(name,prefix,2)==0){const char*map_names[][2]={{"uiRow","row"},{"uiColumn","column"},{"uiContainer","container"},{"uiText","text"},{"uiButton","button"},{"uiInput","input"},{"uiTextarea","textarea"},{"uiSelect","select"},{"uiCheckbox","checkbox"},{"uiRadio","radio"},{"uiTable","table"},{"uiList","list"},{"uiTree","tree"},{"uiTabs","tabs"},{"uiMenu","menu"},{"uiModal","modal"},{"uiTooltip","tooltip"},{"uiProgress","progress"},{"uiImage","image"},{"uiCanvas","canvas"},{"uiSpacer","spacer"},{"uiCustom","custom"}};for(size_t i=0;i<sizeof(map_names)/sizeof(map_names[0]);i++)if(strcmp(name,map_names[i][0])==0){z_expect_args(name,argc,2);return z_ui_value(ZV_UI_NODE,ZK_UI_NODE,z_ui_node_new(map_names[i][1],args[0],args[1]));}}
    if(strcmp(name,"uiMount")==0){z_expect_args(name,argc,4);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);ZDesktopWindowNative*w=z_desktop_expect_window(args[1]);ZUIContextNative*c=(ZUIContextNative*)z_alloc(sizeof(*c));memset(c,0,sizeof(*c));c->app=app;c->window=w;c->root=z_ui_expect_node(args[2]);ZValue theme=z_desktop_dict_get(args[3],"theme");c->theme=theme.tag==ZV_UI_THEME?z_ui_expect_theme(theme):z_ui_theme_new(z_string("light"));c->dirty=true;c->next=app->ui_contexts;app->ui_contexts=c;z_ui_attach_tree(c,c->root);z_ui_render_context(c);return z_ui_value(ZV_UI_CONTEXT,ZK_UI_CONTEXT,c);}
    if(strcmp(name,"uiUnmount")==0){z_expect_args(name,argc,1);ZUIContextNative*c=z_ui_expect_context(args[0]);bool changed=!c->closed;z_ui_sync_text_input(c,NULL);c->focus_id=NULL;c->closed=true;return z_bool(changed);}
    if(strcmp(name,"uiRender")==0){z_expect_args(name,argc,1);z_ui_render_context(z_ui_expect_context(args[0]));return z_bool(true);}
    if(strcmp(name,"uiSnapshot")==0){z_expect_args(name,argc,1);return z_ui_snapshot(z_ui_expect_context(args[0]));}
    if(strcmp(name,"uiSetTheme")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);c->theme=z_ui_expect_theme(args[1]);z_ui_mark_dirty(c);z_ui_render_context(c);return args[0];}
    if(strcmp(name,"uiDispatch")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);if(args[1].tag!=ZV_DICT)z_fatal("uiDispatch event must be dictionary");ZDesktopEventNative e={0};e.type=z_strdup(z_ui_prop_string(args[1],"type","custom"));e.window_id=c->window->id;e.timestamp=z_desktop_now_ns();e.data=args[1];z_ui_dispatch_event(c,&e);z_ui_render_context(c);return z_bool(true);}
    if(strcmp(name,"uiSet")==0){z_expect_args(name,argc,3);ZUINodeNative*n=z_ui_expect_node(args[0]);const char*property=z_as_cstring(args[1]);z_ui_apply_property(n,property,args[2]);if(n->context)z_ui_mark_dirty(n->context);return args[0];}
    if(strcmp(name,"uiGet")==0){z_expect_args(name,argc,2);ZUINodeNative*n=z_ui_expect_node(args[0]);const char*property=z_as_cstring(args[1]);if(strcmp(property,"id")==0)return z_string(n->id);if(strcmp(property,"kind")==0)return z_string(n->kind);return z_desktop_dict_get(n->props,property);}
    if(strcmp(name,"uiAdd")==0){z_expect_args(name,argc,2);ZUINodeNative*p=z_ui_expect_node(args[0]),*child=z_ui_expect_node(args[1]);size_t len=p->children.as.array->len;ZValue*items=(ZValue*)z_alloc(sizeof(ZValue)*(len+1));if(len)memcpy(items,p->children.as.array->items,sizeof(ZValue)*len);items[len]=args[1];p->children=z_array_from(items,len+1);child->parent=p;if(p->context)z_ui_attach_tree(p->context,child);if(p->context)z_ui_mark_dirty(p->context);return args[0];}
    if(strcmp(name,"uiRemove")==0){z_expect_args(name,argc,2);ZUINodeNative*p=z_ui_expect_node(args[0]);const char*id=z_as_cstring(args[1]);size_t len=p->children.as.array->len;ZValue*items=(ZValue*)z_alloc(sizeof(ZValue)*len);size_t count=0;bool removed=false;for(size_t i=0;i<len;i++){ZUINodeNative*c=z_ui_expect_node(p->children.as.array->items[i]);if(strcmp(c->id,id)==0){removed=true;c->parent=NULL;}else items[count++]=p->children.as.array->items[i];}p->children=z_array_from(items,count);if(p->context)z_ui_mark_dirty(p->context);return z_bool(removed);}
    if(strcmp(name,"uiFind")==0){z_expect_args(name,argc,2);ZUINodeNative*n=z_ui_find_node(z_ui_expect_context(args[0])->root,z_as_cstring(args[1]));return n?z_ui_value(ZV_UI_NODE,ZK_UI_NODE,n):z_null();}
    if(strcmp(name,"uiFocus")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);ZUINodeNative*n=z_ui_expect_node(args[1]);if(!z_ui_focusable(n))return z_bool(false);c->focus_id=z_strdup(n->id);z_ui_sync_text_input(c,n);z_ui_mark_dirty(c);return z_bool(true);}
    if(strcmp(name,"uiFocusNext")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);bool reverse=args[1].tag==ZV_BOOL&&args[1].as.b;ZUINodeNative*nodes[2048];size_t count=0;ZUINodeNative*focus_root=z_ui_topmost_modal(c->root);if(!focus_root)focus_root=c->root;z_ui_collect_focusable_nodes(focus_root,nodes,&count,2048);if(count==0)return z_null();size_t current=count;for(size_t i=0;i<count;i++)if(c->focus_id&&strcmp(nodes[i]->id,c->focus_id)==0){current=i;break;}size_t next=current==count?(reverse?count-1:0):(reverse?(current+count-1)%count:(current+1)%count);c->focus_id=z_strdup(nodes[next]->id);z_ui_sync_text_input(c,nodes[next]);z_ui_mark_dirty(c);return z_ui_value(ZV_UI_NODE,ZK_UI_NODE,nodes[next]);}
    if(strcmp(name,"uiAccessibility")==0){z_expect_args(name,argc,1);ZValue items[2048];size_t count=0;ZUIContextNative*context=z_ui_expect_context(args[0]);ZUINodeNative*access_root=z_ui_topmost_modal(context->root);if(!access_root)access_root=context->root;z_ui_collect_accessibility(access_root,items,&count,2048);return z_array_from(items,count);}
    if(strcmp(name,"uiCanvasCommand")==0){z_expect_args(name,argc,3);ZUINodeNative*n=z_ui_expect_node(args[0]);if(strcmp(n->kind,"canvas")!=0)z_fatal("uiCanvasCommand expects canvas");ZValue commands=z_desktop_dict_get(n->props,"commands");size_t len=commands.tag==ZV_ARRAY?commands.as.array->len:0;ZValue*items=(ZValue*)z_alloc(sizeof(ZValue)*(len+1));if(len)memcpy(items,commands.as.array->items,sizeof(ZValue)*len);ZValue pairs[2]={z_pair(z_string("kind"),args[1]),z_pair(z_string("values"),args[2])};items[len]=z_dict_from(pairs,2);z_ui_dict_set(n->props,"commands",z_array_from(items,len+1));if(n->context)z_ui_mark_dirty(n->context);return args[0];}
    if(handled) { *handled=false; }
    return z_null();
}

#endif
