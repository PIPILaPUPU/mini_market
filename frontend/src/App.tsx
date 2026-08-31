import { FormEvent, useState } from "react";
import {
  Link,
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";
import {
  ApiError,
  closeRegistrationSession,
  login,
  register,
} from "./api";

type NoticeState = {
  registrationComplete?: boolean;
  username?: string;
};

function BrandPanel() {
  return (
    <aside className="brand-panel">
      <Link className="brand" to="/" aria-label="Mini Market">
        <span className="brand__mark" aria-hidden="true">
          M
        </span>
        <span>mini market</span>
      </Link>
      <div className="brand-panel__content">
        <p className="eyebrow">Торговая площадка</p>
        <h1>Всё нужное — в одном месте.</h1>
        <p>
          Покупайте и продавайте безопасно. Ваш аккаунт защищён короткими
          сессиями и надёжным шифрованием пароля.
        </p>
      </div>
      <div className="brand-panel__orbit" aria-hidden="true">
        <span />
        <span />
        <span />
      </div>
    </aside>
  );
}

function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <main className="auth-shell">
      <BrandPanel />
      <section className="form-panel">
        <div className="form-panel__inner">{children}</div>
        <p className="security-note">
          <LockIcon />
          Защищённое соединение
        </p>
      </section>
    </main>
  );
}

function LoginPage() {
  const location = useLocation();
  const notice = (location.state as NoticeState | null) ?? {};
  const [username, setUsername] = useState(notice.username ?? "");
  const [password, setPassword] = useState("");
  const [status, setStatus] = useState<"idle" | "loading" | "success">("idle");
  const [error, setError] = useState("");

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setStatus("loading");
    try {
      const response = await login(username, password);
      sessionStorage.setItem("access_token", response.access_token);
      setStatus("success");
    } catch (requestError) {
      setStatus("idle");
      setError(errorMessage(requestError));
    }
  }

  const successful = status === "success";

  return (
    <AuthLayout>
      <div className="form-heading">
        <p className="eyebrow">С возвращением</p>
        <h2>Войдите в аккаунт</h2>
        <p>Введите данные, указанные при регистрации.</p>
      </div>

      {notice.registrationComplete && !successful && (
        <div className="alert alert--info" role="status">
          <CheckIcon />
          Регистрация завершена. Теперь войдите в аккаунт.
        </div>
      )}

      {successful && (
        <div className="alert alert--success" role="status">
          <CheckIcon />
          Вход выполнен успешно. Добро пожаловать, {username}!
        </div>
      )}

      {error && (
        <div className="alert alert--error" role="alert">
          {error}
        </div>
      )}

      <form className="auth-form" onSubmit={handleSubmit}>
        <label>
          <span>Имя пользователя</span>
          <span className={`input-wrap ${successful ? "input-wrap--success" : ""}`}>
            <UserIcon />
            <input
              autoComplete="username"
              value={username}
              onChange={(event) => {
                setUsername(event.target.value);
                setStatus("idle");
              }}
              placeholder="Ваш username"
              required
            />
            {successful && <CheckIcon />}
          </span>
        </label>

        <label>
          <span>Пароль</span>
          <span className={`input-wrap ${successful ? "input-wrap--success" : ""}`}>
            <LockIcon />
            <input
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => {
                setPassword(event.target.value);
                setStatus("idle");
              }}
              placeholder="Введите пароль"
              required
            />
            {successful && <CheckIcon />}
          </span>
        </label>

        <button className="primary-button" disabled={status === "loading"}>
          {status === "loading" ? "Входим…" : successful ? "Вход выполнен" : "Войти"}
          {!successful && <ArrowIcon />}
        </button>
      </form>

      <p className="form-switch">
        Ещё нет аккаунта? <Link to="/register">Зарегистрироваться</Link>
      </p>
    </AuthLayout>
  );
}

function RegisterPage() {
  const navigate = useNavigate();
  const [form, setForm] = useState({
    username: "",
    email: "",
    password: "",
    first_name: "",
    last_name: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  function update(field: keyof typeof form, value: string) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      await register(form);
      await closeRegistrationSession().catch(() => undefined);
      navigate("/", {
        replace: true,
        state: { registrationComplete: true, username: form.username },
      });
    } catch (requestError) {
      setError(errorMessage(requestError));
      setLoading(false);
    }
  }

  return (
    <AuthLayout>
      <div className="form-heading">
        <p className="eyebrow">Новый аккаунт</p>
        <h2>Создайте профиль</h2>
        <p>Это займёт меньше минуты.</p>
      </div>

      {error && (
        <div className="alert alert--error" role="alert">
          {error}
        </div>
      )}

      <form className="auth-form" onSubmit={handleSubmit}>
        <div className="form-row">
          <label>
            <span>Имя</span>
            <span className="input-wrap">
              <input
                autoComplete="given-name"
                value={form.first_name}
                onChange={(event) => update("first_name", event.target.value)}
                placeholder="Иван"
              />
            </span>
          </label>
          <label>
            <span>Фамилия</span>
            <span className="input-wrap">
              <input
                autoComplete="family-name"
                value={form.last_name}
                onChange={(event) => update("last_name", event.target.value)}
                placeholder="Иванов"
              />
            </span>
          </label>
        </div>

        <label>
          <span>Имя пользователя</span>
          <span className="input-wrap">
            <UserIcon />
            <input
              autoComplete="username"
              value={form.username}
              onChange={(event) => update("username", event.target.value)}
              placeholder="ivan_ivanov"
              pattern="[A-Za-z0-9_]{3,32}"
              title="3–32 латинские буквы, цифры или подчёркивания"
              required
            />
          </span>
        </label>

        <label>
          <span>Email</span>
          <span className="input-wrap">
            <MailIcon />
            <input
              type="email"
              autoComplete="email"
              value={form.email}
              onChange={(event) => update("email", event.target.value)}
              placeholder="name@example.com"
              required
            />
          </span>
        </label>

        <label>
          <span>Пароль</span>
          <span className="input-wrap">
            <LockIcon />
            <input
              type="password"
              autoComplete="new-password"
              value={form.password}
              onChange={(event) => update("password", event.target.value)}
              placeholder="Минимум 8 символов"
              minLength={8}
              maxLength={72}
              required
            />
          </span>
        </label>

        <button className="primary-button" disabled={loading}>
          {loading ? "Создаём аккаунт…" : "Зарегистрироваться"}
          {!loading && <ArrowIcon />}
        </button>
      </form>

      <p className="form-switch">
        Уже есть аккаунт? <Link to="/">Войти</Link>
      </p>
    </AuthLayout>
  );
}

function errorMessage(error: unknown) {
  if (!(error instanceof ApiError)) {
    return "Произошла непредвиденная ошибка.";
  }
  switch (error.code) {
    case "invalid_credentials":
      return "Неверное имя пользователя или пароль.";
    case "username_exists":
      return "Это имя пользователя уже занято.";
    case "email_exists":
      return "Этот email уже зарегистрирован.";
    case "invalid_request":
      return "Проверьте правильность заполнения полей.";
    default:
      return error.message;
  }
}

function UserIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm7 8a7 7 0 0 0-14 0" />
    </svg>
  );
}

function LockIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <rect x="5" y="10" width="14" height="10" rx="2" />
      <path d="M8 10V7a4 4 0 0 1 8 0v3" />
    </svg>
  );
}

function MailIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="m4 7 8 6 8-6" />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m5 12 4 4L19 6" />
    </svg>
  );
}

function ArrowIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M5 12h14m-5-5 5 5-5 5" />
    </svg>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
