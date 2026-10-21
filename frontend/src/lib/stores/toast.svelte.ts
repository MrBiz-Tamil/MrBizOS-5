export interface ToastMessage {
	id: string;
	title: string;
	description?: string;
	type: 'success' | 'error' | 'info' | 'warning';
	duration?: number;
}

class ToastStore {
	toasts = $state<ToastMessage[]>([]);

	show(t: Omit<ToastMessage, 'id'>) {
		const id = Math.random().toString(36).substring(2, 9);
		const newToast: ToastMessage = {
			id,
			duration: 4000,
			...t
		};
		this.toasts = [...this.toasts, newToast];

		if (newToast.duration) {
			setTimeout(() => {
				this.dismiss(id);
			}, newToast.duration);
		}
		return id;
	}

	success(title: string, description?: string) {
		return this.show({ title, description, type: 'success' });
	}

	error(title: string, description?: string) {
		return this.show({ title, description, type: 'error' });
	}

	info(title: string, description?: string) {
		return this.show({ title, description, type: 'info' });
	}

	warning(title: string, description?: string) {
		return this.show({ title, description, type: 'warning' });
	}

	dismiss(id: string) {
		this.toasts = this.toasts.filter((item) => item.id !== id);
	}
}

export const toast = new ToastStore();
