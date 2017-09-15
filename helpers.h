#ifndef WEBVIEW_H
#define WEBVIEW_H

#include <gtk-3.0/gtk/gtk.h>
#include <webkit2/webkit2.h>
#include <JavaScriptCore/JavaScript.h>

extern void closeHandler(guint64);
extern void startHandler(guint64);
extern void wvLoadFinished(guint64, char *);
extern void jsSystemMessage(guint64, gint8, char *, double);
extern void snapshotFinished(guint64 id, cairo_surface_t *surface, char * err);
extern char * getSystemScript();

static inline gboolean window_close_cb(GtkWidget *widget, GdkEvent *event, gpointer parent) {
	(void)widget; (void)event;

	closeHandler((guint64)parent);
	return TRUE;
}

static inline gboolean wv_context_menu_cb(WebKitWebView *webview,
						GtkWidget *default_menu,
						WebKitHitTestResult *hit_test_result,
						gboolean triggered_with_keyboard,
						gpointer userdata) {
	(void)webview;
	(void)default_menu;
	(void)hit_test_result;
	(void)triggered_with_keyboard;
	(void)userdata;

	return TRUE;
}

static inline void wv_load_changed_cb(WebKitWebView *wv, WebKitLoadEvent load_event, gpointer parent) {
	switch (load_event) {
		case WEBKIT_LOAD_STARTED:
			break;
		case WEBKIT_LOAD_REDIRECTED:
			break;
		case WEBKIT_LOAD_COMMITTED:
			break;
		case WEBKIT_LOAD_FINISHED:
			wvLoadFinished((guint64) parent, (char *)webkit_web_view_get_uri(wv));
			break;
		}
}

static inline gboolean wv_load_failed_cb(WebKitWebView *wv, WebKitLoadEvent load_event, gchar *failing_uri,
	GError* error, gpointer user_data) {

}
static inline char * js_get_str(JSStringRef sv) {
	gsize len = JSStringGetMaximumUTF8CStringSize (sv);
	gchar *str = (gchar *)g_malloc (len);
	JSStringGetUTF8CString(sv, str, len);
	JSStringRelease(sv);
	return str;
}

static inline void js_system_postmessage(WebKitUserContentManager *m, WebKitJavascriptResult *js_result, gpointer data) {
	JSGlobalContextRef ctx = webkit_javascript_result_get_global_context (js_result);
	JSValueRef val = webkit_javascript_result_get_value (js_result);

	guint8 typ = JSValueGetType(ctx, val);
	double num = 0;
	JSStringRef sv = NULL;

	if(typ == kJSTypeString) {
		sv = JSValueToStringCopy (ctx, val, NULL);
	} else if(typ == kJSTypeBoolean) {
		num = JSValueToBoolean(ctx, val);
	} else if(typ == kJSTypeNumber) {
		num = JSValueToNumber(ctx, val, NULL);
	} else if(typ == kJSTypeObject) {
		sv = JSValueCreateJSONString(ctx, val, 0, NULL);
	}

	if(sv != NULL) {
		char *str = js_get_str(sv);
		jsSystemMessage((guint64)data, typ, str, num);
		g_free(str);
	} else {
		jsSystemMessage((guint64)data, typ, NULL, num);
	}
	webkit_javascript_result_unref (js_result);
}

typedef struct {
	gboolean EnableJava;
	gboolean EnablePlugins;
	gboolean EnableFrameFlattening;
	gboolean EnableSmoothScrolling;
	gboolean EnableSpellChecking;
	gboolean EnableFullscreen;
	gboolean EnableLocalFileAccess;
	gboolean IgnoreTLSErrors;

	gboolean EnableJavaScript;
	gboolean EnableJavaScriptCanOpenWindows;
	gboolean AllowModalDialogs;

	gboolean EnableWriteConsoleMessagesToStdout;

	gboolean EnableWebGL;

	gboolean Decorated;
	gboolean Resizable;

	int Width;
	int Height;

} settings_t;

static inline GtkWidget *create_window(gboolean offscreen) {
	if (gtk_init_check(0, NULL) == FALSE) return NULL;
	if(offscreen) return gtk_offscreen_window_new();
	return gtk_window_new(GTK_WINDOW_TOPLEVEL);
}

static inline WebKitUserContentManager *init_content_manager(guint64 parent) {
	WebKitUserContentManager *cm = webkit_user_content_manager_new();
	g_signal_connect(cm, "script-message-received::system", G_CALLBACK (js_system_postmessage), (void*)parent);
	webkit_user_content_manager_register_script_message_handler (cm, "system");

	char * str = getSystemScript();
	WebKitUserScript *js = webkit_user_script_new(str,
		WEBKIT_USER_CONTENT_INJECT_ALL_FRAMES,
		WEBKIT_USER_SCRIPT_INJECT_AT_DOCUMENT_START,
		NULL, NULL);

	webkit_user_content_manager_add_script(cm, js);
	g_free(str);

	return cm;
}

static inline WebKitWebView *init_window(GtkWidget *window, const char *title, const char *user_agent, settings_t *s, guint64 parent) {
	gtk_widget_hide_on_delete(window);
	gtk_window_set_title(GTK_WINDOW(window), title);
	gtk_window_set_decorated(GTK_WINDOW(window), s->Decorated);


	WebKitSettings *settings = webkit_settings_new();
	webkit_settings_set_enable_java(settings, s->EnableJava);
	webkit_settings_set_enable_plugins(settings, FALSE);
	webkit_settings_set_enable_frame_flattening(settings, s->EnableFrameFlattening);
	webkit_settings_set_enable_smooth_scrolling(settings, s->EnableSmoothScrolling);
	webkit_settings_set_enable_fullscreen(settings, s->EnableFullscreen);
	webkit_settings_set_allow_file_access_from_file_urls(settings, s->EnableLocalFileAccess);


	webkit_settings_set_enable_javascript(settings, s->EnableJavaScript);
	webkit_settings_set_javascript_can_open_windows_automatically(settings, s->EnableJavaScriptCanOpenWindows);
	webkit_settings_set_allow_modal_dialogs(settings, s->AllowModalDialogs);

	webkit_settings_set_enable_write_console_messages_to_stdout(settings, s->EnableWriteConsoleMessagesToStdout);
	webkit_settings_set_enable_webgl (settings, s->EnableWebGL);

	if(user_agent != NULL) webkit_settings_set_user_agent(settings, user_agent);

	if(s->Width == -1 && s->Height == -1) {
		gtk_window_fullscreen(GTK_WINDOW(window));
	} else {
		if (s->Resizable) gtk_window_set_default_size(GTK_WINDOW(window), s->Width, s->Height);
		gtk_window_set_resizable(GTK_WINDOW(window), s->Resizable);
		gtk_widget_set_size_request(window,  s->Width, s->Height);
	}
	gtk_window_set_position(GTK_WINDOW(window), GTK_WIN_POS_CENTER);

	// setup simple messaging from javascript to go.

	WebKitWebView *wv = (WebKitWebView *)webkit_web_view_new_with_user_content_manager(init_content_manager(parent));
	webkit_web_view_set_settings(wv, settings);

	if(s->EnableSpellChecking) {
		webkit_web_context_set_spell_checking_enabled(webkit_web_view_get_context(wv), s->EnableSpellChecking);
	}

	if(s->IgnoreTLSErrors) {
		webkit_web_context_set_tls_errors_policy(webkit_web_view_get_context(wv), WEBKIT_TLS_ERRORS_POLICY_IGNORE);
	}

	g_signal_connect(window, "delete-event", G_CALLBACK(window_close_cb), (void*)parent);
	g_signal_connect(wv, "load-changed", G_CALLBACK(wv_load_changed_cb),  (void*)parent);
	g_signal_connect(wv, "context-menu", G_CALLBACK(wv_context_menu_cb),  (void*)parent);

	GtkWidget *scroller = gtk_scrolled_window_new(NULL, NULL);
	gtk_container_add(GTK_CONTAINER(window), scroller);
	gtk_container_add(GTK_CONTAINER(scroller), GTK_WIDGET(wv));
	startHandler(parent);
	gtk_widget_grab_focus(GTK_WIDGET(wv));
	gtk_widget_show_all(window);

	return wv;
}

static inline void close_window(WebKitWebView *wv, GtkWidget *win) {
	webkit_web_view_try_close(wv);
	gtk_widget_destroy(win);
}

static inline void set_prop(WebKitSettings *s, const char * prop, gboolean v) {
	g_object_set (G_OBJECT(s), prop, v, NULL);
}

#endif /* WEBVIEW_H */
