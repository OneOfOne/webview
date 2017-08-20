#ifndef WEBVIEW_H
#define WEBVIEW_H

#include <gtk/gtk.h>
#include <webkit2/webkit2.h>

extern void close_handler();
extern void start_handler();
extern void wv_load_finished();
extern void wv_load_status_changed();
extern void in_gtk_main(guint64);

static void webview_desroy_cb(GtkWidget *widget, gpointer parent) {
	(void)widget;
	close_handler();
	gtk_main_quit();
}

static gboolean webview_context_menu_cb(WebKitWebView *webview,
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

static void idle_add(guint64 v) {
	g_idle_add((GSourceFunc)in_gtk_main, (gpointer)v);
}

static GtkWidget* window;
static WebKitWebView *webview;


static void load_changed_cb(WebKitWebView *wv, WebKitLoadEvent load_event, gpointer user_data) {
	printf("loading %s %d\n", webkit_web_view_get_uri(webview),  load_event);
	switch (load_event) {
		case WEBKIT_LOAD_STARTED:
			break;
		case WEBKIT_LOAD_REDIRECTED:
			break;
		case WEBKIT_LOAD_COMMITTED:
			break;
		case WEBKIT_LOAD_FINISHED:
			// wv_load_finished();
			break;
		}
}

static WebKitSettings * const defaultSettings(const char *user_agent) {
	WebKitSettings *settings = webkit_settings_new();
	webkit_settings_set_enable_java(settings, false);
	webkit_settings_set_enable_javascript(settings, true);
	webkit_settings_set_enable_plugins(settings, false);
	webkit_settings_set_enable_frame_flattening(settings, true);
	webkit_settings_set_user_agent(settings, user_agent);
	webkit_settings_set_enable_smooth_scrolling(settings, true);
	webkit_settings_set_javascript_can_open_windows_automatically(settings, true);

	return settings;
}
static void create_window(const char *user_agent) {
	if (gtk_init_check(0, NULL) == FALSE) return;

	window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
	g_signal_connect(G_OBJECT(window), "destroy", G_CALLBACK(webview_desroy_cb), NULL);

	webview = (WebKitWebView *)webkit_web_view_new_with_settings(defaultSettings(user_agent));

	g_signal_connect(webview, "load-changed", G_CALLBACK(load_changed_cb), NULL);
	g_signal_connect(webview, "context-menu", G_CALLBACK(webview_context_menu_cb), NULL);


	GtkWidget *scroller = gtk_scrolled_window_new(NULL, NULL);
	gtk_container_add(GTK_CONTAINER(window), scroller);
	gtk_container_add(GTK_CONTAINER(scroller), GTK_WIDGET(webview));

	start_handler();
	gtk_main();

}

static void setWebView(const char *title, const char *url, int width, int height, int resizable) {
	gtk_window_set_title(GTK_WINDOW(window), title);

	if (resizable) {
		gtk_window_set_default_size(GTK_WINDOW(window), width, height);
	}

	gtk_widget_set_size_request(window, width, height);
	gtk_window_set_resizable(GTK_WINDOW(window), !!resizable);
	gtk_window_set_position(GTK_WINDOW(window), GTK_WIN_POS_CENTER);

	webkit_web_view_load_uri(WEBKIT_WEB_VIEW(webview), url);
	gtk_widget_show_all(window);
}

#endif /* WEBVIEW_H */
