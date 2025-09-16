ssh css-g88-002 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20006  10.155.84.1 &
ssh css-g88-002 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20007  10.155.84.1 &
ssh css-g88-003 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20008  10.155.84.1 &
ssh css-g88-003 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20009  10.155.84.1 &
ssh css-g88-004 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20010  10.155.84.1 &
ssh css-g88-004 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20011  10.155.84.1 &
