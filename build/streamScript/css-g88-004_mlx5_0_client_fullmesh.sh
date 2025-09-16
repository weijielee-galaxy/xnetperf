ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20036  10.155.84.4 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20037  10.155.84.4 &
ssh css-g88-002 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20038  10.155.84.4 &
ssh css-g88-002 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20039  10.155.84.4 &
ssh css-g88-003 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20040  10.155.84.4 &
ssh css-g88-003 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20041  10.155.84.4 &
