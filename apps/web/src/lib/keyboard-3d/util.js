// @ts-nocheck
export function toRad(degrees) {
	return (degrees * Math.PI) / 180;
}

export function getKeyRow(index, layout) {
	return Math.floor(layout[index].y);
}

export function getKeyProfile(index, layout, total_rows) {
	total_rows = total_rows || 5;
	let row = Math.floor(layout[index].y);
	if (total_rows < 5) {
		//40%
		row++;
	} else if (total_rows > 5) {
		//75%+
		row = row === 0 ? 1 : row;
		row = row > 4 ? 4 : row;
	} else {
		//60/65
		row++;
		row = row > 4 ? 4 : row;
	}
	return row;
}
