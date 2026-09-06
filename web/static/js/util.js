
// 날짜 리턴 한국방식
function formatDate(dateStr) {
	const date = new Date(dateStr);
	if (isNaN(date)) return "";

	const y = date.getFullYear();
	const m = String(date.getMonth() + 1).padStart(2, "0");
	const d = String(date.getDate()).padStart(2, "0");

	let hour = date.getHours();
	const min = String(date.getMinutes()).padStart(2, "0");
	const sec = String(date.getSeconds()).padStart(2, "0");
	const isPM = hour >= 12;

	const period = isPM ? "오후" : "오전";
	if (hour === 0) hour = 12;
	else if (hour > 12) hour -= 12;

	return `${y}년 ${m}월 ${d}일 ${period} ${hour}시 ${min}분 ${sec}초`;
}


// 숫자를 세 자리마다 쉼표 넣기
function formatNumber(n) {
	return Number(n).toLocaleString("ko-KR");
}

// 문자열 앞뒤 공백 제거하고 null 처리
function cleanString(str) {
	return typeof str === "string" ? str.trim() || null : null;
}


window.util = {
	formatDate,
	formatNumber,
	cleanString
};